// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
)

type DB struct {
	dsn  string
	pool *pgxpool.Pool
}

func New(dsn string) *DB {
	return &DB{
		dsn: dsn,
	}
}

func (d *DB) Init() error {
	pool, err := pgxpool.Connect(context.Background(), d.dsn)
	if err != nil {
		return fmt.Errorf("could not initialize pool: %w", err)
	}
	d.pool = pool
	return migrateUp(d.dsn)
}

func (d *DB) Connection() (*pgxpool.Conn, error) {
	if d.pool == nil {
		return nil, errors.New("pool not initialized")
	}
	conn, err := d.pool.Acquire(context.Background())
	if err != nil {
		return nil, fmt.Errorf("could not acquire connection: %w", err)
	}
	return conn, nil
}

func (d *DB) Close() {
	if d.pool != nil {
		d.pool.Close()
	}
}
