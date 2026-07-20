// Station registry — the data sources behind the conditions. Served verbatim by
// the /stations endpoint.
package main

// Station is a data source we pull ocean/weather readings from. DepthMeters is
// the water depth at the station, used to solve the wave-length dispersion
// relation for that spot.
type Station struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	Lat         float64 `json:"lat"`
	Long        float64 `json:"long"`
	DepthMeters float64 `json:"depthMeters"`
	ProfileURL  string  `json:"profileUrl"`
	DetailsURL  string  `json:"detailsUrl"`
}

// stationMVCO / stationBuoy are the registry codes. The buoy code matches
// vineyardBuoyID.
const (
	stationMVCO = "MVCO"
	stationBuoy = vineyardBuoyID
)

// stations is the fixed registry.
var stations = []Station{
	{
		Code:        stationMVCO,
		Name:        "Martha's Vineyard Coastal Observatory",
		Lat:         41.325,
		Long:        -70.5667,
		DepthMeters: 4,
		ProfileURL:  "https://mvco.whoi.edu/wp-content/uploads/2023/05/MVCO-overview.png",
		DetailsURL:  "https://mvco.whoi.edu/",
	},
	{
		Code:        stationBuoy,
		Name:        "Nantucket Sound Buoy",
		Lat:         41.497,
		Long:        -70.283,
		DepthMeters: 16.5,
		ProfileURL:  "https://www.ndbc.noaa.gov/images/stations/3mfoam_scoop_mini.jpg",
		DetailsURL:  "https://www.ndbc.noaa.gov/station_page.php?station=44020",
	},
}

// stationDepth returns the water depth (m) for a station code, or nil if the
// code isn't in the registry (falls back to the deep-water wave-length calc).
func stationDepth(code string) *float64 {
	for _, s := range stations {
		if s.Code == code {
			d := s.DepthMeters
			return &d
		}
	}
	return nil
}
