package main

import (
	"fmt"
	"math"
	"testing"
)

func TestBumpyScoreSweep(t *testing.T) {
	graphHeightCurve()

	gradyWhiteFreedom215 := boat{Length: 6.55, Weight: 1429}

	waveHeights := []float64{0.3, 0.6, 1.2, 2}
	wavelengths := []float64{30, 40, 80}
	waveDirections := []float64{15}
	windSpeeds := []float64{0, 5, 10, 30}

	for _, wh := range waveHeights {
		for _, wl := range wavelengths {
			for _, wd := range waveDirections {
				for _, ws := range windSpeeds {
					score := calculateDaBumpyScore(&wh, &wl, &wd, &ws, gradyWhiteFreedom215)
					fmt.Printf("height=%.1f length=%.0f direction=%s ws=%.0f -> score=%v\n",
						wh, wl, *calculateCardinalDirection(&wd), ws, *score)
				}
			}
		}
		fmt.Println("-----------")
	}
}

func graphHeightCurve() {
	maxWaveHeight := 5.0
	maxScore := 0.0

	var heights []float64
	steps := int(maxWaveHeight / 0.3)
	for i := 1; i <= steps; i++ {
		h := float64(i) * 0.3
		heights = append(heights, h)
	}

	scores := make(map[float64]float64)
	for _, h := range heights {
		s := math.Pow(h, heightExponent) * heightMultiplier
		scores[h] = s
		if s > maxScore {
			maxScore = s
		}
	}

	for _, h := range heights {
		s := scores[h]
		barLen := int((s / maxScore) * 60)
		bar := ""
		for i := 0; i < barLen; i++ {
			bar += "#"
		}
		fmt.Printf("%4.1fm | %s %.1f\n", h, bar, s)
	}
}
