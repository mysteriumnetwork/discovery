// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// +build ignore

package main

import (
	"os"

	"github.com/magefile/mage/mage"
)

// Zero install option.
// Usage example:
//   go run mage.go test
func main() { os.Exit(mage.Main()) }
