// Station registry — the data sources behind the conditions. Served verbatim by
// the /stations endpoint.
package main

// Station is a data source we pull ocean/weather readings from.
type Station struct {
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	Lat        float64 `json:"lat"`
	Long       float64 `json:"long"`
	ProfileURL string  `json:"profileUrl"`
	DetailsURL string  `json:"detailsUrl"`
}

// stations is the fixed registry. The buoy code matches vineyardBuoyID.
var stations = []Station{
	{
		Code:       "MVCO",
		Name:       "Martha's Vineyard Coastal Observatory",
		Lat:        41.325,
		Long:       -70.5667,
		ProfileURL: "https://mvco.whoi.edu/wp-content/uploads/2023/05/MVCO-overview.png",
		DetailsURL: "https://mvco.whoi.edu/",
	},
	{
		Code:       "44020",
		Name:       "Nantucket Sound Buoy",
		Lat:        41.497,
		Long:       -70.283,
		ProfileURL: "https://www.ndbc.noaa.gov/images/stations/3mfoam_scoop_mini.jpg",
		DetailsURL: "https://www.ndbc.noaa.gov/station_page.php?station=44020",
	},
}
