package e2e

import (
	_ "embed"
	"encoding/csv"
	"strings"
)

//go:embed proposals.csv
var proposalsCSV string

var ProposalsCSVRecords = func() ([][]string, error) {
	return csv.NewReader(strings.NewReader(proposalsCSV)).ReadAll()
}
