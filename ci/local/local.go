// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package local

import (
	"time"

	"github.com/magefile/mage/sh"
)

func Up() {
	err := sh.RunV("docker-compose", "up", "-d", "db")
	if err != nil {
		return
	}
	time.Sleep(3 * time.Second)
	defer sh.RunV("docker-compose", "down", "-v")
	envs := map[string]string{
		"DB_DSN":             "postgresql://discovery:discovery@localhost:5432/discovery",
		"QUALITY_ORACLE_URL": "https://testnet2-quality.mysterium.network",
		"BROKER_URL":         "nats://testnet2-broker.mysterium.network",
	}
	_ = sh.RunWith(envs, "go", "run", "cmd/main.go")
}
