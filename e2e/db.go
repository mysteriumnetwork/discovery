package e2e

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
)

func purgeProposalsDB() error {
	pool, err := pgxpool.Connect(context.Background(), DBdsn)
	if err != nil {
		return fmt.Errorf("could not initialize pool: %w", err)
	}
	defer pool.Close()

	conn, err := pool.Acquire(context.Background())
	if err != nil {
		return fmt.Errorf("could not acquire connection: %w", err)
	}
	defer conn.Release()
	a, err := conn.Exec(context.Background(), "DELETE FROM proposals")
	if err != nil {
		return fmt.Errorf("could not delete proposals from db: %w", err)
	}
	fmt.Printf("successfully deleted %d proposals\n", a.RowsAffected())
	return nil
}
