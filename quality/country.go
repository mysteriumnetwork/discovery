package quality

import (
	_ "embed"
	"strings"
)

//https://dev.maxmind.com/geoip/legacy/codes/iso3166/
//go:embed countries.csv
var countryCSV string

type CountryProvider interface {
	Countries() []string
}

type csvCountryProvider struct {
	countries []string
}

func NewCSVCountryProvider() *csvCountryProvider {
	lines := strings.Split(countryCSV, "\n")
	countries := make([]string, len(lines))
	for idx, line := range lines {
		countries[idx] = strings.Split(line, ",")[0]
	}
	return &csvCountryProvider{countries: countries}
}

func (cp *csvCountryProvider) Countries() []string {
	return cp.countries
}
