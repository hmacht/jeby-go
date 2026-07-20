package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	// Embed the timezone database so America/New_York resolves even on minimal
	// container images that ship without system tzdata.
	_ "time/tzdata"
)

// defaultBumpyScoreRefresh is the fallback cadence when the env override is
// unset or invalid.
const defaultBumpyScoreRefresh = time.Hour

// bumpyScoreRefreshInterval is how often we recompute the BumpyScore. Override
// with BUMPY_SCORE_REFRESH_MINUTES (whole minutes); unset, non-numeric, or
// non-positive values fall back to the default.
func bumpyScoreRefreshInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv("BUMPY_SCORE_REFRESH_MINUTES"))
	if raw == "" {
		return defaultBumpyScoreRefresh
	}
	mins, err := strconv.Atoi(raw)
	if err != nil || mins <= 0 {
		log.Printf("bumpy score worker: invalid BUMPY_SCORE_REFRESH_MINUTES %q, using default %s", raw, defaultBumpyScoreRefresh)
		return defaultBumpyScoreRefresh
	}
	return time.Duration(mins) * time.Minute
}

// Quiet hours: we skip AI analysis at night (local Massachusetts time), from
// quietHoursStart until quietHoursEnd.
const (
	quietHoursStart = 21 // 9pm
	quietHoursEnd   = 5  // 5am
)

// quietHoursMessage is surfaced in the served result while AI analysis is paused.
const quietHoursMessage = "In quiet hours, reading come back online at 5am EST"

// quietHoursResult is what we serve overnight: no score, with the quiet-hours
// message in both the analysis and disclaimers.
func quietHoursResult() bumpyScoreResult {
	msg := quietHoursMessage
	return bumpyScoreResult{
		Score:       nil,
		Analysis:    &msg,
		Disclaimers: []string{quietHoursMessage},
	}
}

// quietHoursResults is the quiet-hours result applied to every vessel, so the
// store stays keyed by vessel code even overnight.
func quietHoursResults() map[string]bumpyScoreResult {
	r := quietHoursResult()
	m := make(map[string]bumpyScoreResult, len(vessels))
	for _, v := range vessels {
		m[v.Code] = r
	}
	return m
}

// massachusettsTZ is loaded once. If it can't load, quiet hours are disabled
// (inQuietHours returns false) rather than silently skipping at the wrong times.
var massachusettsTZ = loadMassachusettsTZ()

func loadMassachusettsTZ() *time.Location {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		log.Printf("bumpy score worker: could not load America/New_York (%v); quiet hours disabled", err)
		return nil
	}
	return loc
}

// inQuietHours reports whether now falls in the nightly no-AI window in
// Massachusetts local time (handles EST/EDT).
func inQuietHours(now time.Time) bool {
	if massachusettsTZ == nil {
		return false
	}
	h := now.In(massachusettsTZ).Hour()
	return h >= quietHoursStart || h < quietHoursEnd
}

// bumpyScoreWorkerDisabled is the kill switch: set BUMPY_SCORE_WORKER_DISABLED to 1/true/yes
// (case-insensitive) to keep the background worker from starting.
func bumpyScoreWorkerDisabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("BUMPY_SCORE_WORKER_DISABLED"))) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}

// bumpyStore holds the latest AI-computed BumpyScores, keyed by vessel code.
// It's read by the request handlers and written by the background worker, so
// it's mutex-guarded.
type bumpyStore struct {
	mu        sync.RWMutex
	results   map[string]bumpyScoreResult
	updatedAt time.Time
	ok        bool
}

func newBumpyStore() *bumpyStore {
	return &bumpyStore{}
}

// get returns the latest result for a vessel code and whether one is available
// (a refresh has run and produced a score for that code).
func (s *bumpyStore) get(code string) (bumpyScoreResult, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.ok {
		return bumpyScoreResult{}, false
	}
	r, exists := s.results[code]
	return r, exists
}

func (s *bumpyStore) set(results map[string]bumpyScoreResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.results = results
	s.updatedAt = time.Now()
	s.ok = true
}

// startBumpyScoreWorker refreshes the BumpyScore once immediately, then on every
// tick, until ctx is cancelled. Reuses the existing MVCO fetch + parse.
func startBumpyScoreWorker(ctx context.Context, ai AIClient, store *bumpyStore, interval time.Duration) {
	// run gates a refresh on quiet hours so we don't burn AI calls overnight.
	run := func(trigger string) {
		if inQuietHours(time.Now()) {
			log.Printf("bumpy score worker [%s]: quiet hours (%dpm-%dam Massachusetts time), serving quiet-hours message", trigger, quietHoursStart-12, quietHoursEnd)
			store.set(quietHoursResults())
			return
		}
		refresh(ctx, ai, store, trigger)
	}

	go func() {
		log.Printf("bumpy score worker: started, refreshing every %s (quiet %dpm-%dam Massachusetts time)", interval, quietHoursStart-12, quietHoursEnd)
		run("boot")

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Print("bumpy score worker: stopping (context cancelled)")
				return
			case <-ticker.C:
				run("tick")
			}
		}
	}()
}

// refresh does one fetch -> parse -> AI score -> store cycle. Failures are
// logged and the previous score is left in place. trigger labels the cycle
// ("boot" or "tick") so you can follow along in the logs.
func refresh(ctx context.Context, ai AIClient, store *bumpyStore, trigger string) {
	start := time.Now()
	log.Printf("bumpy score worker [%s]: refresh starting", trigger)

	log.Printf("bumpy score worker [%s]: fetching MVCO data", trigger)
	data, err := fetchMvcoData()
	if err != nil {
		log.Printf("bumpy score worker [%s]: fetch: %v", trigger, err)
		return
	}
	if data == nil {
		log.Printf("bumpy score worker [%s]: no mvco data", trigger)
		return
	}

	log.Printf("bumpy score worker [%s]: parsing latest reading", trigger)
	reading, err := parseMvcoRecentReading(*data)
	if err != nil {
		log.Printf("bumpy score worker [%s]: parse: %v", trigger, err)
		return
	}

	// Pull the buoy alongside MVCO so the AI can cross-check the two stations.
	// buildBuoyStationConditions logs its own failures and degrades to empty
	// readings, so a buoy outage doesn't stop us from scoring on MVCO alone.
	log.Printf("bumpy score worker [%s]: fetching buoy data", trigger)
	buoy := buildBuoyStationConditions()

	log.Printf("bumpy score worker [%s]: reading at %s, asking AI for BumpyScores", trigger, reading.Time.Format(time.RFC3339))

	results, err := calculateVineyardSoundBumpyScore(ctx, ai, reading, buoy, vessels)
	if err != nil {
		log.Printf("bumpy score worker [%s]: score: %v", trigger, err)
		return
	}

	store.set(results)
	log.Printf("bumpy score worker [%s]: updated %d vessel BumpyScores (took %s)", trigger, len(results), time.Since(start).Round(time.Millisecond))
}
