// Bunch of rando functions I use
package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ptr returns a pointer to v. Handy for the many optional *float64 fields.
func ptr[T any](v T) *T {
	return &v
}

// Given a number and how many decimals you got, we will round for you
func round(v float64, decimals int) float64 {
	multiplier := math.Pow(10, float64(decimals))
	return math.Round(v*multiplier) / multiplier
}

func fmtPtr(f *float64) string {
	if f == nil {
		return "nil"
	}
	return fmt.Sprintf("%.2f", *f)
}

func parseFloat(s string) *float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}
