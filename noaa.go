package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type noaaAlerts struct {
	Title    string `json:"title"`
	Features []struct {
		Properties struct {
			Event    string `json:"event"`
			Headline string `json:"headline"`
			Severity string `json:"severity"`
		} `json:"properties"`
	} `json:"features"`
}

func fetchActiveAlerts(forecastZone string) ([]alert, error) {
	body, err := fetchURL(fmt.Sprintf("https://api.weather.gov/alerts/active?zone=%s", forecastZone))
	if err != nil {
		return nil, fmt.Errorf("fetching alerts: %w", err)
	}

	var noaaAlerts noaaAlerts
	if err := json.Unmarshal(body, &noaaAlerts); err != nil {
		return nil, fmt.Errorf("parsing alerts: %w", err)
	}

	alerts := make([]alert, 0, len(noaaAlerts.Features))
	for _, f := range noaaAlerts.Features {
		alerts = append(alerts, alert{
			Event:       f.Properties.Event,
			Description: f.Properties.Headline,
			Severity:    f.Properties.Severity,
		})
	}
	return alerts, nil
}

func fetchURL(url string) ([]byte, error) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "(jeby.com, team@jeby.com)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
