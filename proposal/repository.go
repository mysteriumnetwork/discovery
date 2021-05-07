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

	"github.com/mysteriumnetwork/discovery/db"
	v2 "github.com/mysteriumnetwork/discovery/proposal/v2"
)

var ctx = context.Background()

type Repository struct {
	expirationJobDelay time.Duration
	expirationDuration time.Duration
	db                 *db.DB
}

func NewRepository(db *db.DB) *Repository {
	return &Repository{
		expirationDuration: 3*time.Minute + 10*time.Second,
		expirationJobDelay: 20 * time.Second,
		db:                 db,
	}
}

type repoListOpts struct {
	providerID                string
	serviceType               string
	country                   string
	ipType                    string
	accessPolicy              string
	accessPolicySource        string
	compatibilityMin          int
	compatibilityMax          int
	priceGiBMax, priceHourMax int64
}

func (r *Repository) List(opts repoListOpts) ([]v2.Proposal, error) {
	conn, err := r.db.Connection()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	var args []interface{}
	//start := time.Now()
	q := strings.Builder{}
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
	if opts.priceGiBMax > 0 {
		args = append(args, opts.priceGiBMax)
		q.WriteString(fmt.Sprintf(" AND (proposal->'price'->>'per_gib')::decimal <= $%v", len(args)))
	}
	if opts.priceHourMax > 0 {
		args = append(args, opts.priceHourMax)
		q.WriteString(fmt.Sprintf(" AND (proposal->'price'->>'per_hour')::decimal <= $%v", len(args)))
	}
	rows, _ := conn.Query(context.Background(), q.String(), args...)
	defer rows.Close()
	//log.Info().Msgf("select: %s", time.Since(start))

	var proposals []v2.Proposal
	for rows.Next() {
		var rp v2.Proposal
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

func (r *Repository) Store(proposal v2.Proposal) error {
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
		INSERT INTO proposals (proposal, key, expires_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (key) DO UPDATE
			SET proposal = $1,
				expires_at = $3;
		`,
		proposalJSON, proposal.Key(), expiresAt.UTC(),
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
