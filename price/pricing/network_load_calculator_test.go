package pricing

import (
	"testing"
)

func TestNetworkLoadMultiplierCalculator_calculateMultiplier(t *testing.T) {
	type args struct {
		providers      uint64
		activeSessions uint64
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{
			name: "expects 1",
			args: args{
				providers:      1,
				activeSessions: 1,
			},
			want: 1,
		},
		{
			name: "expects 1 if 0 providers",
			args: args{
				providers:      0,
				activeSessions: 1,
			},
			want: 1,
		},
		{
			name: "coeff truncated if below 0.5",
			args: args{
				providers:      33,
				activeSessions: 1,
			},
			want: 0.5,
		},
		{
			name: "coeff truncated if above 2",
			args: args{
				providers:      2,
				activeSessions: 1231312312,
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nlmc := &NetworkLoadMultiplierCalculator{}
			if got := nlmc.calculateMultiplier(tt.args.providers, tt.args.activeSessions); got != tt.want {
				t.Errorf("NetworkLoadMultiplierCalculator.calculateMultiplier() = %v, want %v", got, tt.want)
			}
		})
	}
}
