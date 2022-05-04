// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package swagger

import (
	"os"

	"github.com/fatih/color"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/mysteriumnetwork/go-ci/util"
)

// Installs the swag generation tool
func swagInstall() error {
	err := sh.RunV("go", "install", "github.com/swaggo/swag/cmd/swag@v1.7.8")
	if err != nil {
		color.Red("Could not go get swag")
		return err
	}
	return nil
}

// Swag generates the swagger documents
func Swag() error {
	mg.Deps(swagInstall)
	swag, err := util.GetGoBinaryPath("swag")
	if err != nil {
		color.Red("Could not find swag")
		return err
	}

	color.Cyan("Removing old docs")
	err = os.RemoveAll("./docs")
	if err != nil {
		return err
	}
	color.Cyan("Old docs removed")

	color.Cyan("Generating swagger docs with %s", swag)
	err = sh.RunV(
		swag, "init",
		"--generalInfo", "main.go",
		"--dir", "cmd",
		"--parseDependency",
	)
	if err != nil {
		return err
	}
	color.Green("Generating docs DONE!")
	return nil
}
