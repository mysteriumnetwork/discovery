// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package proposal

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	v2 "github.com/mysteriumnetwork/discovery/proposal/v2"
)

var ctx = context.Background()

type Repository struct {
	expirationJobDelay time.Duration
	expirationDuration time.Duration
	pool               *pgxpool.Pool
}

func NewRepository(dbPool *pgxpool.Pool) *Repository {
	return &Repository{
		expirationDuration: 2 * time.Minute,
		expirationJobDelay: 20 * time.Second,
		pool:               dbPool,
	}
}

type repoListOpts struct {
	serviceType, country string
	residential          bool
}

func (r *Repository) List(opts repoListOpts) ([]v2.Proposal, error) {
	conn, err := r.pool.Acquire(context.Background())
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	//start := time.Now()
	rows, _ := conn.Query(context.Background(), `
		SELECT proposal FROM proposals
		WHERE
			($1 = '' OR proposal->>'service_type' = $1)
			AND ($2 = '' OR proposal->'location'->>'country' = $2)
			AND ($3 IS NOT TRUE OR proposal->'location'->>'ip_type' = 'residential')
	`, opts.serviceType, opts.country, opts.residential)
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

	conn, err := r.pool.Acquire(context.Background())
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
		proposalJSON, proposal.Key(), expiresAt,
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) Expire() (int64, error) {
	conn, err := r.pool.Acquire(ctx)
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
