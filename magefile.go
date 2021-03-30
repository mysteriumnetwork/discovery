// +build mage

package main

import (
	"path"

	"github.com/magefile/mage/sh"
	"github.com/mysteriumnetwork/discovery/ci/local"
)

// Build builds the app binary.
//goland:noinspection GoUnusedExportedFunction
func Build() error {
	return sh.Run("go", "build", "-o", path.Join("build", "discovery"), path.Join("cmd", "main.go"))
}

// Run runs the app (without the DB).
//goland:noinspection GoUnusedExportedFunction
func Run() error {
	return sh.RunV("go", "run", "./cmd/main.go")
}

// Up runs the discovery stack (app and DB) locally.
//goland:noinspection GoUnusedExportedFunction
func Up() {
	local.Up()
}
