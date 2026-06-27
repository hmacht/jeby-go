package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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

// Fetches the NDBC (National Data Buoy Center) realtime buoy data
func fetchRealtimeBuoyData(buoyId string) (string, error) {
	body, err := fetchURL(fmt.Sprintf("https://www.ndbc.noaa.gov/data/realtime2/%s.txt", buoyId))

	if err != nil {
		return "", fmt.Errorf("fetching alerts: %w", err)
	}

	return string(body), nil
}

// This gets the Marine text forecast. Its just a big text block of upcoming days text
func fetchMarineForcastSummary(forcastZoneId string) (forcastSummary, error) {
	body, err := fetchURL(fmt.Sprintf("https://tgftp.nws.noaa.gov/data/forecasts/marine/coastal/an/%s.txt", strings.ToLower(forcastZoneId)))

	if err != nil {
		return forcastSummary{}, fmt.Errorf("fetching marine forecast summary: %w", err)
	}

	forcast, err := parseForecastSummary(string(body))
	if err != nil {
		return forcastSummary{}, fmt.Errorf("parsing marine forecast summary: %w", err)
	}

	return forcast, nil
}

// This gets any images we have at sea from a given buoy
func fetchBuoyImages(buoyId string) (buoyImageData, error) {
	baseUrl := "https://www.ndbc.noaa.gov"
	body, err := fetchURL(fmt.Sprintf("%s/station_page.php?station=%s", baseUrl, buoyId))

	if err != nil {
		return buoyImageData{}, fmt.Errorf("fetching marine forecast summary: %w", err)
	}

	images, err := parseBuoyWebpage(string(body), baseUrl)
	if err != nil {
		return buoyImageData{}, fmt.Errorf("parsing marine forecast summary: %w", err)
	}

	return images, nil
}

// Shared general function to fetch NOAA API and get body
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

	if resp.StatusCode < 200 || resp.StatusCode > 300 {
		s := string(body)
		fields := strings.Fields(s)
		s = strings.Join(fields, " ")
		s = strings.ReplaceAll(s, `"`, "")
		s = strings.ReplaceAll(s, `\`, "")
		return nil, fmt.Errorf("unexpected status %d from %s: %s", resp.StatusCode, url, s)
	}

	return body, nil
}
