package pricing

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/mysteriumnetwork/payments/crypto"
)

func Test_calculatePrice(t *testing.T) {
	type args struct {
		mystPriceUSD float64
		basePriceUSD float64
		multiplier   float64
	}
	tests := []struct {
		name string
		args args
		want *big.Int
	}{
		{
			name: "calculates correctly 1",
			args: args{
				mystPriceUSD: 0.6,
				basePriceUSD: 0.06,
				multiplier:   1,
			},
			want: crypto.FloatToBigMyst(0.1),
		},
		{
			name: "calculates correctly 2",
			args: args{
				mystPriceUSD: 0.06,
				basePriceUSD: 0.06,
				multiplier:   1,
			},
			want: crypto.FloatToBigMyst(1),
		},
		{
			name: "calculates correctly 3",
			args: args{
				mystPriceUSD: 0.6,
				basePriceUSD: 0.06,
				multiplier:   0.1,
			},
			want: big.NewInt(10000000000000002),
		},
		{
			name: "calculates correctly 4",
			args: args{
				mystPriceUSD: 0.6,
				basePriceUSD: 0.06,
				multiplier:   1.5,
			},
			want: big.NewInt(150000000000000032),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calculatePriceMYST(tt.args.mystPriceUSD, tt.args.basePriceUSD, tt.args.multiplier); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("calculatePriceMYST() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPricer_isMystInSensibleLimit(t *testing.T) {
	type fields struct {
		mystBound Bound
	}
	type args struct {
		price float64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "rejects price that's too high",
			fields: fields{
				mystBound: Bound{
					Min: 0.1,
					Max: 0.2,
				},
			},
			args: args{
				price: 1,
			},
			wantErr: true,
		},
		{
			name: "rejects price that's too low",
			fields: fields{
				mystBound: Bound{
					Min: 0.1,
					Max: 0.2,
				},
			},
			args: args{
				price: 0.01,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pricer{
				mystBound: tt.fields.mystBound,
			}
			if err := p.withinBounds(tt.args.price); (err != nil) != tt.wantErr {
				t.Errorf("Pricer.withinBounds() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPricer_generateNewDefaults(t *testing.T) {
	type fields struct {
		cfg Config
		lp  LatestPrices
	}
	type args struct {
		mystInUSD float64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *PriceHistory
	}{
		{
			name: "Fills previous with current if no previous lp",
			fields: fields{
				lp: LatestPrices{},
				cfg: Config{
					BasePrices: PriceByTypeUSD{
						Residential: &PriceUSD{
							PricePerHour: 0.01,
							PricePerGiB:  0.02,
						},
						Other: &PriceUSD{
							PricePerHour: 0.03,
							PricePerGiB:  0.04,
						},
					},
				},
			},
			args: args{
				mystInUSD: 1,
			},
			want: &PriceHistory{
				Current: &PriceByType{
					Residential: &Price{
						PricePerHour:              crypto.FloatToBigMyst(0.01),
						PricePerHourHumanReadable: 0.01,
						PricePerGiB:               crypto.FloatToBigMyst(0.02),
						PricePerGiBHumanReadable:  0.02,
					},
					Other: &Price{
						PricePerHour:              crypto.FloatToBigMyst(0.03),
						PricePerHourHumanReadable: 0.03,
						PricePerGiB:               crypto.FloatToBigMyst(0.04),
						PricePerGiBHumanReadable:  0.04,
					},
				},
				Previous: &PriceByType{
					Residential: &Price{
						PricePerHour:              crypto.FloatToBigMyst(0.01),
						PricePerHourHumanReadable: 0.01,
						PricePerGiB:               crypto.FloatToBigMyst(0.02),
						PricePerGiBHumanReadable:  0.02,
					},
					Other: &Price{
						PricePerHour:              crypto.FloatToBigMyst(0.03),
						PricePerHourHumanReadable: 0.03,
						PricePerGiB:               crypto.FloatToBigMyst(0.04),
						PricePerGiBHumanReadable:  0.04,
					},
				},
			},
		},
		{
			name: "Fills previous with current if previous lp exists",
			fields: fields{
				lp: LatestPrices{
					Defaults: &PriceHistory{
						Current: &PriceByType{
							Residential: &Price{
								PricePerHour:              crypto.FloatToBigMyst(0.05),
								PricePerHourHumanReadable: 0.05,
								PricePerGiB:               crypto.FloatToBigMyst(0.06),
								PricePerGiBHumanReadable:  0.06,
							},
							Other: &Price{
								PricePerHour:              crypto.FloatToBigMyst(0.07),
								PricePerHourHumanReadable: 0.07,
								PricePerGiB:               crypto.FloatToBigMyst(0.08),
								PricePerGiBHumanReadable:  0.08,
							},
						},
					},
				},
				cfg: Config{
					BasePrices: PriceByTypeUSD{
						Residential: &PriceUSD{
							PricePerHour: 0.01,
							PricePerGiB:  0.02,
						},
						Other: &PriceUSD{
							PricePerHour: 0.03,
							PricePerGiB:  0.04,
						},
					},
				},
			},
			args: args{
				mystInUSD: 1,
			},
			want: &PriceHistory{
				Current: &PriceByType{
					Residential: &Price{
						PricePerHour:              crypto.FloatToBigMyst(0.01),
						PricePerHourHumanReadable: 0.01,
						PricePerGiB:               crypto.FloatToBigMyst(0.02),
						PricePerGiBHumanReadable:  0.02,
					},
					Other: &Price{
						PricePerHour:              crypto.FloatToBigMyst(0.03),
						PricePerHourHumanReadable: 0.03,
						PricePerGiB:               crypto.FloatToBigMyst(0.04),
						PricePerGiBHumanReadable:  0.04,
					},
				},
				Previous: &PriceByType{
					Residential: &Price{
						PricePerHour:              crypto.FloatToBigMyst(0.05),
						PricePerHourHumanReadable: 0.05,
						PricePerGiB:               crypto.FloatToBigMyst(0.06),
						PricePerGiBHumanReadable:  0.06,
					},
					Other: &Price{
						PricePerHour:              crypto.FloatToBigMyst(0.07),
						PricePerHourHumanReadable: 0.07,
						PricePerGiB:               crypto.FloatToBigMyst(0.08),
						PricePerGiBHumanReadable:  0.08,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pricer{
				lp: tt.fields.lp,
			}
			if got := p.generateNewDefaults(tt.args.mystInUSD, tt.fields.cfg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Pricer.generateNewDefaults() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPricer_generateNewPerCountry(t *testing.T) {
	type fields struct {
		cfg Config
		lp  LatestPrices
	}
	type args struct {
		mystInUSD float64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[string]*PriceHistory
	}{
		{
			name: "Fills previous with current if no previous lp exists",
			fields: fields{
				lp: LatestPrices{},
				cfg: Config{
					BasePrices: PriceByTypeUSD{
						Residential: &PriceUSD{
							PricePerHour: 0.01,
							PricePerGiB:  0.02,
						},
						Other: &PriceUSD{
							PricePerHour: 0.03,
							PricePerGiB:  0.04,
						},
					},
					CountryModifiers: map[ISO3166CountryCode]Modifier{
						"US": {
							Residential: 2,
							Other:       3,
						},
					},
				},
			},
			args: args{
				mystInUSD: 1,
			},
			want: map[string]*PriceHistory{
				"US": {
					Current: &PriceByType{
						Residential: &Price{
							PricePerHour:              calculatePriceMYST(1, 0.01, 2),
							PricePerHourHumanReadable: calculatePriceMystFloat(1, 0.01, 2),
							PricePerGiB:               calculatePriceMYST(1, 0.02, 2),
							PricePerGiBHumanReadable:  calculatePriceMystFloat(1, 0.02, 2),
						},
						Other: &Price{
							PricePerHour:              calculatePriceMYST(1, 0.03, 3),
							PricePerHourHumanReadable: calculatePriceMystFloat(1, 0.03, 3),
							PricePerGiB:               calculatePriceMYST(1, 0.04, 3),
							PricePerGiBHumanReadable:  calculatePriceMystFloat(1, 0.04, 3),
						},
					},
					Previous: &PriceByType{
						Residential: &Price{
							PricePerHour:              calculatePriceMYST(1, 0.01, 2),
							PricePerHourHumanReadable: calculatePriceMystFloat(1, 0.01, 2),
							PricePerGiB:               calculatePriceMYST(1, 0.02, 2),
							PricePerGiBHumanReadable:  calculatePriceMystFloat(1, 0.02, 2),
						},
						Other: &Price{
							PricePerHour:              calculatePriceMYST(1, 0.03, 3),
							PricePerHourHumanReadable: calculatePriceMystFloat(1, 0.03, 3),
							PricePerGiB:               calculatePriceMYST(1, 0.04, 3),
							PricePerGiBHumanReadable:  calculatePriceMystFloat(1, 0.04, 3),
						},
					},
				},
			},
		},
		{
			name: "Fills previous with current if previous lp exists",
			fields: fields{
				lp: LatestPrices{
					Defaults: &PriceHistory{},
					PerCountry: map[string]*PriceHistory{
						"US": {
							Current: &PriceByType{
								Residential: &Price{
									PricePerHour:              crypto.FloatToBigMyst(0.05),
									PricePerHourHumanReadable: 0.05,
									PricePerGiB:               crypto.FloatToBigMyst(0.06),
									PricePerGiBHumanReadable:  0.06,
								},
								Other: &Price{
									PricePerHour:              crypto.FloatToBigMyst(0.07),
									PricePerHourHumanReadable: 0.07,
									PricePerGiB:               crypto.FloatToBigMyst(0.08),
									PricePerGiBHumanReadable:  0.08,
								},
							},
						},
					},
				},
				cfg: Config{
					BasePrices: PriceByTypeUSD{
						Residential: &PriceUSD{
							PricePerHour: 0.01,
							PricePerGiB:  0.02,
						},
						Other: &PriceUSD{
							PricePerHour: 0.03,
							PricePerGiB:  0.04,
						},
					},
					CountryModifiers: map[ISO3166CountryCode]Modifier{
						"US": {
							Residential: 2,
							Other:       3,
						},
					},
				},
			},
			args: args{
				mystInUSD: 1,
			},
			want: map[string]*PriceHistory{
				"US": {
					Current: &PriceByType{
						Residential: &Price{
							PricePerHour:              calculatePriceMYST(1, 0.01, 2),
							PricePerHourHumanReadable: calculatePriceMystFloat(1, 0.01, 2),
							PricePerGiB:               calculatePriceMYST(1, 0.02, 2),
							PricePerGiBHumanReadable:  calculatePriceMystFloat(1, 0.02, 2),
						},
						Other: &Price{
							PricePerHour:              calculatePriceMYST(1, 0.03, 3),
							PricePerHourHumanReadable: calculatePriceMystFloat(1, 0.03, 3),
							PricePerGiB:               calculatePriceMYST(1, 0.04, 3),
							PricePerGiBHumanReadable:  calculatePriceMystFloat(1, 0.04, 3),
						},
					},
					Previous: &PriceByType{
						Residential: &Price{
							PricePerHour:              crypto.FloatToBigMyst(0.05),
							PricePerHourHumanReadable: 0.05,
							PricePerGiB:               crypto.FloatToBigMyst(0.06),
							PricePerGiBHumanReadable:  0.06,
						},
						Other: &Price{
							PricePerHour:              crypto.FloatToBigMyst(0.07),
							PricePerHourHumanReadable: 0.07,
							PricePerGiB:               crypto.FloatToBigMyst(0.08),
							PricePerGiBHumanReadable:  0.08,
						},
					},
				},
			},
		},
		{
			name: "Fills previous with current if previous lp exists but has no country",
			fields: fields{
				lp: LatestPrices{
					Defaults:   &PriceHistory{},
					PerCountry: map[string]*PriceHistory{},
				},
				cfg: Config{
					BasePrices: PriceByTypeUSD{
						Residential: &PriceUSD{
							PricePerHour: 0.01,
							PricePerGiB:  0.02,
						},
						Other: &PriceUSD{
							PricePerHour: 0.03,
							PricePerGiB:  0.04,
						},
					},
					CountryModifiers: map[ISO3166CountryCode]Modifier{
						"US": {
							Residential: 2,
							Other:       3,
						},
					},
				},
			},
			args: args{
				mystInUSD: 1,
			},
			want: map[string]*PriceHistory{
				"US": {
					Current: &PriceByType{
						Residential: &Price{
							PricePerHour:              calculatePriceMYST(1, 0.01, 2),
							PricePerHourHumanReadable: calculatePriceMystFloat(1, 0.01, 2),
							PricePerGiB:               calculatePriceMYST(1, 0.02, 2),
							PricePerGiBHumanReadable:  calculatePriceMystFloat(1, 0.02, 2),
						},
						Other: &Price{
							PricePerHour:              calculatePriceMYST(1, 0.03, 3),
							PricePerHourHumanReadable: calculatePriceMystFloat(1, 0.03, 3),
							PricePerGiB:               calculatePriceMYST(1, 0.04, 3),
							PricePerGiBHumanReadable:  calculatePriceMystFloat(1, 0.04, 3),
						},
					},
					Previous: &PriceByType{
						Residential: &Price{
							PricePerHour:              calculatePriceMYST(1, 0.01, 2),
							PricePerHourHumanReadable: calculatePriceMystFloat(1, 0.01, 2),
							PricePerGiB:               calculatePriceMYST(1, 0.02, 2),
							PricePerGiBHumanReadable:  calculatePriceMystFloat(1, 0.02, 2),
						},
						Other: &Price{
							PricePerHour:              calculatePriceMYST(1, 0.03, 3),
							PricePerHourHumanReadable: calculatePriceMystFloat(1, 0.03, 3),
							PricePerGiB:               calculatePriceMYST(1, 0.04, 3),
							PricePerGiBHumanReadable:  calculatePriceMystFloat(1, 0.04, 3),
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pricer{
				loadByCountryProvider: &mockNetworkLoadByCountryProvider{},
				lp:                    tt.fields.lp,
			}
			generated := p.generateNewPerCountry(tt.args.mystInUSD, tt.fields.cfg)
			got := generated["US"]
			want := tt.want["US"]

			if !reflect.DeepEqual(got, want) {
				t.Errorf("Pricer.generateNewPerCountry() = %v, want %v", got, want)
			}
		})
	}
}

type mockNetworkLoadByCountryProvider struct {
}

func (m *mockNetworkLoadByCountryProvider) GetMultiplier(isoCode ISO3166CountryCode) float64 {
	return 1
}
