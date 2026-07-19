package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// The BumpyScore™ is our 0-100 measure of how rough a ride a boat
// will have. This variant asks an AI to reason over both live stations — the
// MVCO offshore reading and the NOAA buoy — rather than running the hand-tuned
// formula in marine_calculations.go.
//
// We are doing the calculation tailored for Vineyard Sound right now. In the
// future we will take into account locations.
const bumpyScoreSystemPrompt = `You are a marine conditions expert for Vineyard Sound, Massachusetts.
You are given two live data sources — the MVCO (Martha's Vineyard Coastal Observatory) offshore station and
the nearby NOAA buoy — plus a list of vessels. Estimate a "BumpyScore" for EACH vessel: an integer from
0 (glassy calm) to 100 (dangerously rough) describing how rough that specific vessel's ride will be in the
current conditions.

Reason over BOTH stations together. They sample slightly different spots, so treat them as corroborating
readings: when they agree you can be confident, and when they disagree, lean toward the rougher picture. If
there is a large discrepancy between the two data sets (for example wave height or wind speed differing
substantially), call that out in the analysis and note it in the disclaimers.

The same seas feel very different from boat to boat: larger, heavier, longer vessels ride more smoothly, and
more horsepower helps hold course and speed. Weigh each vessel's length, weight, horsepower, and passenger
capacity against significant wave height and period, then swell vs wind waves, then wind speed, then trends
implied by pressure. Northerly waves in the Sound are rougher than their height suggests.

Dont specify the vessels details in the analysis, just refrence it by name. For the larger vessele, Steamship,
Island Queen, and Large Crafts, recomend outside or inside or if its calm enough and dosent matter you can specify that too.

Respond with ONLY a JSON object, no prose and no markdown fences, in exactly this shape:
{"vessels":[{"code":<vessel code>,"score":<integer 0-100>,"analysis":<plain-language summary for this vessel, 3 sentences max>,"disclaimers":[<short strings noting assumptions or missing data>]}]}
Include exactly one entry for every vessel code listed below.`

// aiBumpyScores is the JSON contract we ask the model to return: one score per
// vessel.
type aiBumpyScores struct {
	Vessels []aiVesselScore `json:"vessels"`
}

type aiVesselScore struct {
	Code        string   `json:"code"`
	Score       *int     `json:"score"`
	Analysis    *string  `json:"analysis"`
	Disclaimers []string `json:"disclaimers"`
}

// calculateVineyardSoundBumpyScore asks the AI to score the current conditions
// for every vessel, reasoning over both the MVCO offshore station and the NOAA
// buoy. Returns a map keyed by vessel code.
func calculateVineyardSoundBumpyScore(ctx context.Context, ai AIClient, mvco MvcoReading, buoy StationConditions, vessels []Vessel) (map[string]bumpyScoreResult, error) {
	prompt := describeMvcoReading(mvco) + "\n" + describeBuoyReadings(buoy) + "\nVessels:\n" + describeVessels(vessels)
	raw, err := ai.Message(ctx, AIMessageRequest{
		System:    bumpyScoreSystemPrompt,
		Prompt:    prompt,
		MaxTokens: 8192,
	})
	if err != nil {
		return nil, fmt.Errorf("ai bumpy score: %w", err)
	}

	parsed, err := parseAIBumpyScores(raw)
	if err != nil {
		return nil, fmt.Errorf("parsing ai bumpy scores from %q: %w", raw, err)
	}

	results := make(map[string]bumpyScoreResult, len(parsed.Vessels))
	for _, v := range parsed.Vessels {
		results[v.Code] = bumpyScoreResult{
			Score:       v.Score,
			Analysis:    v.Analysis,
			Disclaimers: v.Disclaimers,
		}
	}
	return results, nil
}

// describeVessels renders the vessel registry into a labeled list the model can
// weigh each score against.
func describeVessels(vessels []Vessel) string {
	var b strings.Builder
	for _, v := range vessels {
		fmt.Fprintf(&b, "- %s (%s): length %s, weight %s, horsepower %s, max passengers %s\n",
			v.Code, v.Name, v.Length, v.Weight, v.Horsepower, v.MaxPassengers)
	}
	return b.String()
}

// describeMvcoReading renders the reading into a compact, labeled block the
// model can reason over. Missing sensors show as "nil".
func describeMvcoReading(r MvcoReading) string {
	windCardinal := "nil"
	if c := calculateCardinalDirection(r.WindDirectionMean); c != nil {
		windCardinal = *c
	}

	var b strings.Builder
	fmt.Fprintf(&b, "MVCO reading at %s (Vineyard Sound):\n", r.Time.Format("2006-01-02 15:04 MST"))

	fmt.Fprintf(&b, "Waves:\n")
	fmt.Fprintf(&b, "- Significant wave height (m): %s\n", fmtPtr(r.WaveHeightSig))
	fmt.Fprintf(&b, "- Dominant wave period (s): %s\n", fmtPtr(r.WavePeriodDom))
	fmt.Fprintf(&b, "- Swell wave height (m): %s, swell period (s): %s\n", fmtPtr(r.WaveHeightSwell), fmtPtr(r.WavePeriodSwell))
	fmt.Fprintf(&b, "- Wind-wave height (m): %s, wind-wave period (s): %s\n", fmtPtr(r.WaveHeightWind), fmtPtr(r.WavePeriodWind))
	fmt.Fprintf(&b, "- Water level / tide (m): %s\n", fmtPtr(r.WaterLevel))

	fmt.Fprintf(&b, "Wind (anemometer):\n")
	fmt.Fprintf(&b, "- Wind direction (deg from): %s (%s)\n", fmtPtr(r.WindDirectionMean), windCardinal)
	fmt.Fprintf(&b, "- Wind speed mean/median (m/s): %s / %s\n", fmtPtr(r.WindSpeedMean), fmtPtr(r.WindSpeedMedian))
	fmt.Fprintf(&b, "- Vertical wind speed mean (m/s): %s\n", fmtPtr(r.WindSpeedWMean))
	fmt.Fprintf(&b, "- Wind gustiness, std of u/v/w components (m/s): %s / %s / %s\n", fmtPtr(r.WindSpeedUStd), fmtPtr(r.WindSpeedVStd), fmtPtr(r.WindSpeedWStd))
	fmt.Fprintf(&b, "- Sonic air temp mean/median (C): %s / %s\n", fmtPtr(r.SonicTempMean), fmtPtr(r.SonicTempMedian))

	fmt.Fprintf(&b, "Atmosphere:\n")
	fmt.Fprintf(&b, "- Pressure mean/median/std (mmHg): %s / %s / %s\n", fmtPtr(r.PressureMean), fmtPtr(r.PressureMedian), fmtPtr(r.PressureStd))
	fmt.Fprintf(&b, "- Air temp mean/median/std (C): %s / %s / %s\n", fmtPtr(r.TempMean), fmtPtr(r.TempMedian), fmtPtr(r.TempStd))
	fmt.Fprintf(&b, "- Humidity mean/median/std (%%): %s / %s / %s\n", fmtPtr(r.HumidityMean), fmtPtr(r.HumidityMedian), fmtPtr(r.HumidityStd))

	return b.String()
}

// describeBuoyReadings renders the averaged NOAA buoy conditions so the model can
// cross-check them against the MVCO reading. It's the second data source the
// scorer weighs; large gaps between the two stations are worth flagging.
func describeBuoyReadings(b StationConditions) string {
	windCardinal := "nil"
	if b.WindDirectionCardinal != nil {
		windCardinal = *b.WindDirectionCardinal
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "NOAA buoy %s reading (Nantucket Sound), recent average:\n", vineyardBuoyID)
	fmt.Fprintf(&sb, "- Significant wave height (m): %s\n", fmtPtr(b.WaveHeight.Value))
	fmt.Fprintf(&sb, "- Average wave period (s): %s\n", fmtPtr(b.WavePeriod.Value))
	fmt.Fprintf(&sb, "- Wave length (m): %s\n", fmtPtr(b.WaveLength.Value))
	fmt.Fprintf(&sb, "- Wind direction (deg from): %s (%s)\n", fmtPtr(b.WindDirection.Value), windCardinal)
	fmt.Fprintf(&sb, "- Wind speed (m/s): %s\n", fmtPtr(b.WindSpeed.Value))
	fmt.Fprintf(&sb, "- Water temp (C): %s\n", fmtPtr(b.WaterTemp.Value))

	return sb.String()
}

// parseAIBumpyScores pulls the JSON object out of the model's response and
// validates every vessel score.
func parseAIBumpyScores(raw string) (aiBumpyScores, error) {
	start := strings.IndexByte(raw, '{')
	end := strings.LastIndexByte(raw, '}')
	if start < 0 || end < start {
		return aiBumpyScores{}, fmt.Errorf("no JSON object found")
	}

	var out aiBumpyScores
	if err := json.Unmarshal([]byte(raw[start:end+1]), &out); err != nil {
		return aiBumpyScores{}, err
	}
	if len(out.Vessels) == 0 {
		return aiBumpyScores{}, fmt.Errorf("response contained no vessel scores")
	}
	for _, v := range out.Vessels {
		if v.Code == "" {
			return aiBumpyScores{}, fmt.Errorf("vessel score missing code")
		}
		if v.Score == nil {
			return aiBumpyScores{}, fmt.Errorf("vessel %s missing score", v.Code)
		}
		if *v.Score < 0 || *v.Score > 100 {
			return aiBumpyScores{}, fmt.Errorf("vessel %s score %d out of range 0-100", v.Code, *v.Score)
		}
	}
	return out, nil
}
