// Bunch of rando functions I use
package main

import (
	"math"
)

// Given a number and how many decimals you got, we will round for you
func round(v float64, decimals int) float64 {
	multiplier := math.Pow(10, float64(decimals))
	return math.Round(v*multiplier) / multiplier
}
