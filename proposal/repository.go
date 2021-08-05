// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package proposal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/mysteriumnetwork/discovery/db"
	v3 "github.com/mysteriumnetwork/discovery/proposal/v3"
)

var ctx = context.Background()

type Repository struct {
	expirationJobDelay time.Duration
	expirationDuration time.Duration
	db                 *db.DB
	enhancers          []Enhancer
}

type Enhancer interface {
	Enhance(proposal *v3.Proposal)
}

func NewRepository(db *db.DB, enhancers []Enhancer) *Repository {
	return &Repository{
		expirationDuration: 3*time.Minute + 10*time.Second,
		expirationJobDelay: 20 * time.Second,
		db:                 db,
		enhancers:          enhancers,
	}
}

type repoListOpts struct {
	providerID         string
	serviceType        string
	country            string
	ipType             string
	accessPolicy       string
	accessPolicySource string
	compatibilityMin   int
	compatibilityMax   int
	tags               string
}

func (r *Repository) List(opts repoListOpts) ([]v3.Proposal, error) {
	q := strings.Builder{}
	var args []interface{}

	q.WriteString("SELECT proposal FROM proposals WHERE 1=1")
	if opts.providerID != "" {
		args = append(args, opts.providerID)
		q.WriteString(fmt.Sprintf(" AND proposal->>'provider_id' = $%v", len(args)))
	}
	if opts.serviceType != "" {
		args = append(args, opts.serviceType)
		q.WriteString(fmt.Sprintf(" AND proposal->>'service_type' = $%v", len(args)))
	}
	if opts.country != "" {
		args = append(args, opts.country)
		q.WriteString(fmt.Sprintf(" AND proposal->'location'->>'country' = $%v", len(args)))
	}
	if opts.compatibilityMin == 0 && opts.compatibilityMax == 0 {
		// defaults, ignore and return all
	} else if opts.compatibilityMin == opts.compatibilityMax {
		args = append(args, fmt.Sprint(opts.compatibilityMax))
		q.WriteString(fmt.Sprintf(" AND proposal->>'compatibility' = $%v", len(args)))
	} else {
		args = append(args, opts.compatibilityMin)
		q.WriteString(fmt.Sprintf(" AND (proposal->>'compatibility')::int >= $%v", len(args)))

		args = append(args, opts.compatibilityMax)
		q.WriteString(fmt.Sprintf(" AND (proposal->>'compatibility')::int <= $%v", len(args)))
	}
	if opts.ipType != "" {
		args = append(args, opts.ipType)
		q.WriteString(fmt.Sprintf(" AND proposal->'location'->>'ip_type' = $%v", len(args)))
	}
	if opts.accessPolicy == "" {
		q.WriteString(" AND proposal ? 'access_policies' = FALSE")
	} else if opts.accessPolicy == "all" {
		// all access policies
	} else {
		q.WriteString(fmt.Sprintf(` AND proposal->'access_policies' @> '[{"id": "%s"}]'`, opts.accessPolicy))
	}
	if opts.accessPolicySource != "" {
		q.WriteString(fmt.Sprintf(` AND proposal->'access_policies' @> '[{"source": "%s"}]'`, opts.accessPolicySource))
	}
	if opts.tags != "" {
		splits := strings.Split(opts.tags, ",")
		for i := range splits {
			splits[i] = fmt.Sprintf("'%v'", splits[i])
		}
		q.WriteString(fmt.Sprintf(` AND proposal->'tags' ?| ARRAY[%v]`, strings.Join(splits, ",")))
	}

	conn, err := r.db.Connection()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	rows, _ := conn.Query(context.Background(), q.String(), args...)
	defer rows.Close()
	//log.Info().Msgf("select: %s", time.Since(start))

	var proposals []v3.Proposal
	for rows.Next() {
		var rp v3.Proposal
		if err := rows.Scan(&rp); err != nil {
			return nil, err
		}
		proposals = append(proposals, rp)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return proposals, nil
}

type repoMetadataOpts struct {
	providerID string
}

func (r *Repository) Metadata(opts repoMetadataOpts) ([]v3.Metadata, error) {
	q := strings.Builder{}
	var args []interface{}

	q.WriteString(`
        SELECT proposal->>'provider_id'                                             AS provider_id,
               proposal->>'service_type'                                            AS service_type,
               proposal->'location'->>'country'                                     AS country,
               proposal->'location'->>'isp'                                         AS isp,
               proposal->'location'->>'ip_type'                                     AS ip_type,
               COALESCE(proposal->'access_policies'@>'[{"id":"mysterium"}]', FALSE) AS whitelist,
               updated_at                                                           AS updated_at
        FROM proposals
        WHERE 1=1
	`)
	if opts.providerID != "" {
		args = append(args, opts.providerID)
		q.WriteString(fmt.Sprintf(" AND proposal->>'provider_id' = $%v", len(args)))
	}

	conn, err := r.db.Connection()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	rows, _ := conn.Query(context.Background(), q.String(), args...)
	defer rows.Close()

	var meta []v3.Metadata
	for rows.Next() {
		var m v3.Metadata
		if err := pgxscan.ScanRow(&m, rows); err != nil {
			return nil, err
		}
		meta = append(meta, m)
	}
	return meta, nil
}

func (r *Repository) enhance(proposal *v3.Proposal) {
	for i := range r.enhancers {
		r.enhancers[i].Enhance(proposal)
	}
}

func (r *Repository) Store(proposal v3.Proposal) error {
	r.enhance(&proposal)
	expiresAt := time.Now().Add(r.expirationDuration)

	proposalJSON, err := json.Marshal(proposal)
	if err != nil {
		return err
	}

	conn, err := r.db.Connection()
	if err != nil {
		return err
	}
	defer conn.Release()

	_, err = conn.Exec(context.Background(), `
		INSERT INTO proposals (proposal, key, expires_at, updated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (key) DO UPDATE
			SET proposal = $1,
				expires_at = $3,
				updated_at = $4;
		`,
		proposalJSON, proposal.Key(), expiresAt.UTC(), time.Now().UTC(),
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) Expire() (int64, error) {
	conn, err := r.db.Connection()
	if err != nil {
		return 0, err
	}
	defer conn.Release()

	cmd, err := conn.Exec(ctx, `
		DELETE FROM proposals
		WHERE expires_at < now()
	`)
	if err != nil {
		return 0, err
	}
	return cmd.RowsAffected(), nil
}

func (r *Repository) Remove(key string) (int64, error) {
	conn, err := r.db.Connection()
	if err != nil {
		return 0, err
	}
	defer conn.Release()

	cmd, err := conn.Exec(ctx, `
			DELETE FROM proposals
			WHERE key = $1
		`, key)
	if err != nil {
		return 0, err
	}
	return cmd.RowsAffected(), nil
}
