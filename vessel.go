// Vessel registry — the boats we compute per-vessel BumpyScores for. The specs
// feed the AI prompt and are also served verbatim by the /vessels endpoint.
package main

// Vessel is a boat we score. Physical specs are free-form strings because the
// craft categories are ranges ("26-65 ft") and some specs are unknown.
type Vessel struct {
	Code          string `json:"code"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Weight        string `json:"weight"`
	Length        string `json:"length"`
	Horsepower    string `json:"horsepower"`
	MaxPassengers string `json:"maxPassengers"`
}

// vessels is the fixed registry. Order is preserved in the /vessels response and
// in the AI prompt. The three named boats are specific; SMALL/MEDIUM/LARGE are
// generic size classes described by ranges.
var vessels = []Vessel{
	{Code: "SSA", Name: "Steamship", Description: "Large Steamship Authority car-and-passenger ferry, it can carry 18 wheelers!.", Weight: "unknown", Length: "255 ft", Horsepower: "6,000 HP", MaxPassengers: "1,210"},
	{Code: "IQ", Name: "Island Queen", Description: "The queen herself. Passenger ferry running between Falmouth and Oak Bluffs. Serves drinks and candy at the bar! Front bow is fun for all ages.", Weight: "99 tons", Length: "125 ft", Horsepower: "1,000 HP", MaxPassengers: "522"},
	{Code: "F215", Name: "Grady White Freedom 215", Description: "21.5 ft Freedom 215 duel console. Its long heavy hull slices the swells differntly than lets say a 17' Boston Whaler Montauk", Weight: "3,150 lb", Length: "21.5 ft", Horsepower: "150 HP", MaxPassengers: "8"},
	{Code: "SMALL", Name: "Small Craft", Description: "Small recreational craft such as skiffs, runabouts, and small center consoles.", Weight: "under 5,000 lb", Length: "under 26 ft", Horsepower: "25-300 HP", MaxPassengers: "4-8"},
	{Code: "MEDIUM", Name: "Medium Craft", Description: "Mid-size craft such as cabin cruisers and larger center consoles.", Weight: "5,000-40,000 lb", Length: "26-65 ft", Horsepower: "200-1,500 HP", MaxPassengers: "8-49"},
	{Code: "LARGE", Name: "Large Craft", Description: "Large craft such as yachts and commercial vessels.", Weight: "40,000+ lb", Length: "65 ft and up", Horsepower: "1,000-10,000+ HP", MaxPassengers: "12-150+"},
}

// vesselByCode looks up a vessel by its code.
func vesselByCode(code string) (Vessel, bool) {
	for _, v := range vessels {
		if v.Code == code {
			return v, true
		}
	}
	return Vessel{}, false
}
