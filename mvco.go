package main

import (
	"fmt"
)

// MVCO Provides a CSV of the live reading
// Reading come in about every 20 mins
func fetchMvcoData() (*string, error) {
	body, err := fetchURL("https://mvco.whoi.edu/aux/plots/front-page-data.csv")
	if err != nil {
		return nil, fmt.Errorf("fetching mvco data: %w", err)
	}
	s := string(body)
	return &s, nil
}
