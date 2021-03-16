package v2

import "math/big"

// Currency represents a supported currency
type Currency string

// MystSize represents a size of the Myst token.
var MystSize = big.NewInt(1_000_000_000_000_000_000)

const (
	// CurrencyMyst is the myst token currency representation
	CurrencyMyst = Currency("MYST")

	// CurrencyMystt is the test myst token currency representation
	CurrencyMystt = Currency("MYSTT")
)
