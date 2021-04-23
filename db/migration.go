// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package db

import (
	"embed"

	"github.com/golang-migrate/migrate/v4"
	pgdriver "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/rs/zerolog/log"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func migrateUp(dsn string) error {
	log.Info().Msg("Running DB migrations")
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return err
	}
	db := stdlib.OpenDB(*cfg)
	driver, err := pgdriver.WithInstance(db, &pgdriver.Config{
		DatabaseName: cfg.Database,
	})
	if err != nil {
		return err
	}

	sqlMigrationsFS, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return err
	}
	mig, err := migrate.NewWithInstance("iofs", sqlMigrationsFS, cfg.Database, driver)
	if err != nil {
		return err
	}
	err = mig.Up()
	if err == migrate.ErrNoChange {
		log.Info().Msg("Nothing to migrate")
		return nil
	}
	return err
}
