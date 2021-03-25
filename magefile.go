// +build mage

package main

import (
	"path"

	"github.com/magefile/mage/sh"
	"github.com/mysteriumnetwork/discovery/ci/local"
)

// Build builds the project.
func Build() error {
	return sh.Run("go", "build", "-o", path.Join("build", "discovery"), path.Join("cmd", "main.go"))
}

// Run runs the project.
func Run() error {
	return sh.RunV("go", "run", "./cmd/main.go")
}

// Local runs local discovery stack.
func Local() {
	local.Local()
}
