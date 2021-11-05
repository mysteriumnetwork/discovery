// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// +build mage

package main

import (
	"path"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"

	"github.com/mysteriumnetwork/discovery/ci/e2e"
	"github.com/mysteriumnetwork/discovery/ci/local"
	"github.com/mysteriumnetwork/discovery/ci/swagger"
	"github.com/mysteriumnetwork/go-ci/commands"
)

// Swag generates swagger JSON.
func Swag() error {
	return swagger.Swag()
}

// Test runs the tests.
func Test() error {
	return sh.RunV("go", "test", "-race", "-cover", "-short", "./...")
}

// Check checks that the source is compliant with all of the checks.
func Check() error {
	return commands.CheckD(".")
}

func Copyright() error {
	return commands.CopyrightD(".")
}

// Build builds the app binary.
//goland:noinspection GoUnusedExportedFunction
func Build() error {
	mg.Deps(Swag)
	return sh.Run("go", "build", "-o", path.Join("build", "discovery"), path.Join("cmd", "main.go"))
}

// Build builds the sidecar binary.
//goland:noinspection GoUnusedExportedFunction
func BuildSidecar() error {
	return sh.Run("go", "build", "-o", path.Join("build", "sidecar"), path.Join("sidecar", "cmd", "main.go"))
}

// Run runs the app (without the DB).
//goland:noinspection GoUnusedExportedFunction
func Run() error {
	envs := map[string]string{
		"DB_DSN":              "postgresql://discovery:discovery@localhost:5432/discovery",
		"QUALITY_ORACLE_URL":  "https://quality.mysterium.network",
		"BROKER_URL":          "nats://broker.mysterium.network",
		"COINRANKING_TOKEN":   "",
		"UNIVERSE_JWT_SECRET": "",
		"REDIS_ADDRESS":       "localhost:6379",
		"BADGER_ADDRESS":      "https://badger.mysterium.network",
		"LOCATION_ADDRESS":    "https://location.mysterium.network/api/v1/location",
	}
	return sh.RunWithV(envs, "go", "run", "./cmd/main.go")
}

// Up runs the discovery stack (app and DB) locally.
//goland:noinspection GoUnusedExportedFunction
func Up() {
	mg.Deps(Swag)
	local.Up()
}

// E2EDev spins up local NATS and seeded DB for e2e test development.
//goland:noinspection GoUnusedExportedFunction
func E2EDev() {
	e2e.UpDevDependencies()
}

// E2E runs e2e tests on locally running instance
//goland:noinspection GoUnusedExportedFunction
func E2E() {
	e2e.Run()
}
