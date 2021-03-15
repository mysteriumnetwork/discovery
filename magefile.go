// +build mage

package main

import (
	"path"

	"github.com/magefile/mage/sh"
)

// Build builds the project.
func Build() error {
	return sh.Run("go", "build", "-o", path.Join("build", "discovery"), path.Join("cmd", "main.go"))
}
