package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// The BumpyScore™ is our 0-100 measure of how rough a ride a boat
// will have. This variant asks an AI to reason over a live MVCO reading rather
// than running the hand-tuned formula in marine_calculations.go.
//
// We are doing the calculation tailored for Vineyard Sound right now. In the
// future we will take into account locations.
const bumpyScoreSystemPrompt = `You are a marine conditions expert for Vineyard Sound, Massachusetts.
Given a live sensor reading from the MVCO (Martha's Vineyard Coastal Observatory), estimate a "BumpyScore":
an integer from 0 (glassy calm) to 100 (dangerously rough) describing how rough a small recreational boat's
ride will be. Weigh significant wave height and period most heavily, then swell vs wind waves, then wind speed,
then trends implied by pressure. Northerly waves in the Sound are rougher than their height suggests.

Respond with ONLY a JSON object, no prose and no markdown fences, in exactly this shape:
{"score": <integer 0-100>, "disclaimers": [<short strings noting assumptions or missing data>]}`

// aiBumpyScore is the JSON contract we ask the model to return.
type aiBumpyScore struct {
	Score       *int     `json:"score"`
	Disclaimers []string `json:"disclaimers"`
}

// calculateBumpyScoreFromMVCO asks the AI to score a single MVCO reading.
func calculateBumpyScoreFromMVCO(ctx context.Context, ai AIClient, reading MvcoReading) (bumpyScoreResult, error) {
	raw, err := ai.Message(ctx, AIMessageRequest{
		System:    bumpyScoreSystemPrompt,
		Prompt:    describeMvcoReading(reading),
		MaxTokens: 4096,
	})
	if err != nil {
		return bumpyScoreResult{}, fmt.Errorf("ai bumpy score: %w", err)
	}

	parsed, err := parseAIBumpyScore(raw)
	if err != nil {
		return bumpyScoreResult{}, fmt.Errorf("parsing ai bumpy score from %q: %w", raw, err)
	}

	disclaimers := append([]string{"Score estimated by AI from live MVCO data."}, parsed.Disclaimers...)
	return bumpyScoreResult{
		Score:       parsed.Score,
		Disclaimers: disclaimers,
	}, nil
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

// parseAIBumpyScore pulls the JSON object out of the model's response
func parseAIBumpyScore(raw string) (aiBumpyScore, error) {
	start := strings.IndexByte(raw, '{')
	end := strings.LastIndexByte(raw, '}')
	if start < 0 || end < start {
		return aiBumpyScore{}, fmt.Errorf("no JSON object found")
	}

	var out aiBumpyScore
	if err := json.Unmarshal([]byte(raw[start:end+1]), &out); err != nil {
		return aiBumpyScore{}, err
	}
	if out.Score == nil {
		return aiBumpyScore{}, fmt.Errorf("response missing score")
	}
	if *out.Score < 0 || *out.Score > 100 {
		return aiBumpyScore{}, fmt.Errorf("score %d out of range 0-100", *out.Score)
	}
	return out, nil
}
