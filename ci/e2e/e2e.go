package e2e

import (
	"time"

	"github.com/magefile/mage/sh"
	"github.com/mysteriumnetwork/discovery/e2e"
)

func Run() error {
	defer sh.RunV("docker-compose", "-f", e2e.DockerFile, "down", "-v")
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
	return nil
}

func upDependencies() error {
	err := sh.RunV("docker-compose", "-f", e2e.DockerFile, "up", "-d", "dev")
	if err != nil {
		return err
	}
	time.Sleep(3 * time.Second) // wait for DB initialization
	return nil
}

func upApp() error {
	err := sh.RunV("docker-compose", "-f", e2e.DockerFile, "up", "--build", "-d", "discovery")
	if err != nil {
		return err
	}
	return sh.RunV("docker-compose", "-f", e2e.DockerFile, "up", "--build", "-d", "discopricer")
}
