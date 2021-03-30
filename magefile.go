// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// +build mage

package main

import (
	"path"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
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
	return commands.Test("./...")
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

// Run runs the app (without the DB).
//goland:noinspection GoUnusedExportedFunction
func Run() error {
	mg.Deps(Swag)
	return sh.RunV("go", "run", "./cmd/main.go")
}

// Up runs the discovery stack (app and DB) locally.
//goland:noinspection GoUnusedExportedFunction
func Up() {
	mg.Deps(Swag)
	local.Up()
}
