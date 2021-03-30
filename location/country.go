// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package location

import (
	_ "embed"
	"strings"
)

//https://dev.maxmind.com/geoip/legacy/codes/iso3166/
//go:embed countries.csv
var countryCSV string

var Countries = func() []string {
	lines := strings.Split(countryCSV, "\n")
	count := len(lines) - 1 // The last line is empty
	res := make([]string, count)
	for i, line := range lines[:count] {
		res[i] = strings.Split(line, ",")[0]
	}
	return res
}()
