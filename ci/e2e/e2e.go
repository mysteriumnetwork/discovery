package e2e

import (
	"context"
	"time"

	"github.com/magefile/mage/sh"
	"github.com/mysteriumnetwork/discovery/db"
	"github.com/mysteriumnetwork/discovery/e2e"
	"github.com/rs/zerolog/log"
)

const (
	dbDSN      = "postgresql://discovery:discovery@localhost:5432/discovery"
	dockerFile = "e2e/docker-compose.yml"
)

func Run() error {
	defer sh.RunV("docker-compose", "-f", dockerFile, "down", "-v")
	err := UpDevDependencies()
	if err != nil {
		return err
	}

	if err := upApp(); err != nil {
		return err
	}

	return sh.RunV("go",
		"test",
		"-v",
		"-tags=e2e",
		"./e2e",
	)
}

func UpDevDependencies() error {
	if err := upDependencies(); err != nil {
		return err
	}

	if err := seedDBWithProposals(); err != nil {
		return err
	}
	return nil
}

func seedDBWithProposals() error {
	dataSource := db.New(dbDSN)
	defer dataSource.Close()

	if err := dataSource.Init(); err != nil {
		return err
	}

	conn, err := dataSource.Connection()
	if err != nil {
		return err
	}
	defer conn.Release()
	records, err := e2e.ProposalsCSVRecords()
	if err != nil {
		return err
	}

	log.Info().Msg("Seeding proposals...")
	_, err = conn.Exec(context.Background(), "DELETE FROM PROPOSALS")
	if err != nil {
		return err
	}
	for _, r := range records {
		key := r[0]
		proposalJSON := r[1]
		_, err = conn.Exec(context.Background(), `
		INSERT INTO proposals (proposal, key, expires_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (key) DO UPDATE
			SET proposal = $1,
				expires_at = $3;
		`,
			proposalJSON, key, time.Now().Add(200*time.Hour).UTC(),
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func upDependencies() error {
	err := sh.RunV("docker-compose", "-f", dockerFile, "up", "-d", "dev")
	if err != nil {
		return err
	}
	time.Sleep(3 * time.Second) // wait for DB initialization
	return nil
}

func upApp() error {
	return sh.RunV("docker-compose", "-f", dockerFile, "up", "--build", "-d", "discovery")
}
