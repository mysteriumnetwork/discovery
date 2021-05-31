package pricing

import (
	_ "embed"
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	mprice := &PriceUSD{
		PricePerHour: 1,
		PricePerGiB:  2,
	}
	type fields struct {
		BasePrices       PriceByTypeUSD
		CountryModifiers map[ISO3166CountryCode]Modifier
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "accepts valid config",
			fields: fields{
				BasePrices: PriceByTypeUSD{
					Residential: mprice,
					Other:       mprice,
				},
				CountryModifiers: map[ISO3166CountryCode]Modifier{
					"US": {
						Residential: 1,
						Other:       1,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "detects invalid country name",
			fields: fields{
				BasePrices: PriceByTypeUSD{
					Residential: mprice,
					Other:       mprice,
				},
				CountryModifiers: map[ISO3166CountryCode]Modifier{
					"us": {
						Residential: 1,
						Other:       1,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "detects invalid country modifier",
			fields: fields{
				BasePrices: PriceByTypeUSD{
					Residential: mprice,
					Other:       mprice,
				},
				CountryModifiers: map[ISO3166CountryCode]Modifier{
					"US": {
						Residential: -1,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "detects invalid pricing",
			fields: fields{
				BasePrices: PriceByTypeUSD{
					Residential: mprice,
					Other: &PriceUSD{
						PricePerHour: -1,
						PricePerGiB:  2,
					},
				},
				CountryModifiers: map[ISO3166CountryCode]Modifier{
					"US": {
						Residential: 1,
						Other:       1,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "detects unset pricing",
			fields: fields{
				BasePrices: PriceByTypeUSD{
					Residential: mprice,
					Other:       nil,
				},
				CountryModifiers: map[ISO3166CountryCode]Modifier{
					"US": {
						Residential: 1,
						Other:       1,
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Config{
				BasePrices:       tt.fields.BasePrices,
				CountryModifiers: tt.fields.CountryModifiers,
			}
			if err := c.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
