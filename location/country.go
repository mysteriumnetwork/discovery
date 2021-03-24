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
