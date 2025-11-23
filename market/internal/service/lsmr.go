package service

import (
	"math"
)

// calculateCost calculates the cost function C(q) for LSMR
// C(q) = b * ln(sum(exp(q_i / b)))
func calculateCost(quantities []float64, b float64) float64 {
	if b <= 0 {
		return 0 // Should not happen with valid b
	}

	sumExp := 0.0
	maxQ := 0.0

	// Find max quantity to prevent overflow
	if len(quantities) > 0 {
		maxQ = quantities[0]
		for _, q := range quantities {
			if q > maxQ {
				maxQ = q
			}
		}
	}

	for _, q := range quantities {
		sumExp += math.Exp((q - maxQ) / b)
	}

	return b * (math.Log(sumExp) + maxQ/b)
}

// calculatePrice calculates the instantaneous price of an option
// p_i = exp(q_i / b) / sum(exp(q_j / b))
func calculatePrice(quantities []float64, b float64, optionIndex int) float64 {
	if optionIndex < 0 || optionIndex >= len(quantities) {
		return 0
	}
	if b <= 0 {
		return 0
	}

	sumExp := 0.0
	maxQ := 0.0

	// Find max quantity to prevent overflow
	if len(quantities) > 0 {
		maxQ = quantities[0]
		for _, q := range quantities {
			if q > maxQ {
				maxQ = q
			}
		}
	}

	for _, q := range quantities {
		sumExp += math.Exp((q - maxQ) / b)
	}

	return math.Exp((quantities[optionIndex]-maxQ)/b) / sumExp
}

// calculateCostToBuy calculates the cost to buy a specific amount of shares for an option
// Cost = C(q_new) - C(q_old)
func calculateCostToBuy(currentQuantities []float64, b float64, optionIndex int, amountToBuy float64) float64 {
	if optionIndex < 0 || optionIndex >= len(currentQuantities) {
		return 0
	}

	newQuantities := make([]float64, len(currentQuantities))
	copy(newQuantities, currentQuantities)
	newQuantities[optionIndex] += amountToBuy

	return calculateCost(newQuantities, b) - calculateCost(currentQuantities, b)
}

// calculateCostToSell calculates the return from selling a specific amount of shares for an option
// Return = C(q_old) - C(q_new) (which is negative of cost to buy negative amount)
func calculateCostToSell(currentQuantities []float64, b float64, optionIndex int, amountToSell float64) float64 {
	if optionIndex < 0 || optionIndex >= len(currentQuantities) {
		return 0
	}

	newQuantities := make([]float64, len(currentQuantities))
	copy(newQuantities, currentQuantities)
	newQuantities[optionIndex] -= amountToSell

	// Return is positive if we get money back
	return calculateCost(currentQuantities, b) - calculateCost(newQuantities, b)
}
