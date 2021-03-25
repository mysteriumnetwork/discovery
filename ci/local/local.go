package local

import (
	"time"

	"github.com/magefile/mage/sh"
)

func Local() {
	err := sh.RunV("docker-compose", "up", "-d", "db")
	if err != nil {
		return
	}
	time.Sleep(3 * time.Second)
	defer sh.RunV("docker-compose", "down", "-v")
	envs := map[string]string{
		"DB_HOST":            "localhost:6379",
		"DB_PASSWORD":        "",
		"QUALITY_ORACLE_URL": "https://testnet2-quality.mysterium.network",
		"BROKER_URL":         "nats://testnet2-broker.mysterium.network",
	}
	_ = sh.RunWith(envs, "go", "run", "cmd/main.go")
}
