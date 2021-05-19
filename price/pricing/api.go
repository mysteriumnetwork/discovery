package pricing

type MockPriceAPI struct {
}

func (mpapi *MockPriceAPI) MystUSD() (float64, error) {
	return 0.5186, nil
}
