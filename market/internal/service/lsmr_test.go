package service

import (
	"math"
	"testing"
)

func TestCalculateCost(t *testing.T) {
	tests := []struct {
		name       string
		quantities []float64
		b          float64
		want       float64
	}{
		{
			name:       "Zero quantities",
			quantities: []float64{0, 0},
			b:          100.0,
			want:       100.0 * math.Log(2),
		},
		{
			name:       "Equal quantities",
			quantities: []float64{100, 100},
			b:          100.0,
			want:       100.0 * (math.Log(2) + 1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateCost(tt.quantities, tt.b)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("calculateCost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculatePrice(t *testing.T) {
	tests := []struct {
		name        string
		quantities  []float64
		b           float64
		optionIndex int
		want        float64
	}{
		{
			name:        "Zero quantities",
			quantities:  []float64{0, 0},
			b:           100.0,
			optionIndex: 0,
			want:        0.5,
		},
		{
			name:        "Higher quantity for option 0",
			quantities:  []float64{100, 0},
			b:           100.0,
			optionIndex: 0,
			want:        math.Exp(1) / (math.Exp(1) + 1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculatePrice(tt.quantities, tt.b, tt.optionIndex)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("calculatePrice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateCostToBuy(t *testing.T) {
	tests := []struct {
		name              string
		currentQuantities []float64
		b                 float64
		optionIndex       int
		amount            float64
		want              float64
	}{
		{
			name:              "Buy from zero",
			currentQuantities: []float64{0, 0},
			b:                 100.0,
			optionIndex:       0,
			amount:            10.0,
			want:              calculateCost([]float64{10, 0}, 100.0) - calculateCost([]float64{0, 0}, 100.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateCostToBuy(tt.currentQuantities, tt.b, tt.optionIndex, tt.amount)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("calculateCostToBuy() = %v, want %v", got, tt.want)
			}
		})
	}
}
