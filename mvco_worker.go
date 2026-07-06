package main

import (
	"context"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	// Embed the timezone database so America/New_York resolves even on minimal
	// container images that ship without system tzdata.
	_ "time/tzdata"
)

// MvcoRefreshInterval is how often we recompute the BumpyScore. AI analysis is
// hourly for now.
const MvcoRefreshInterval = time.Hour

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

// massachusettsTZ is loaded once. If it can't load, quiet hours are disabled
// (inQuietHours returns false) rather than silently skipping at the wrong times.
var massachusettsTZ = loadMassachusettsTZ()

func loadMassachusettsTZ() *time.Location {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		log.Printf("mvco bumpy worker: could not load America/New_York (%v); quiet hours disabled", err)
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

// mvcoWorkerDisabled is the kill switch: set MVCO_WORKER_DISABLED to 1/true/yes
// (case-insensitive) to keep the background worker from starting.
func mvcoWorkerDisabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("MVCO_WORKER_DISABLED"))) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}

// bumpyStore holds the latest AI-computed BumpyScore. It's read by the request
// handlers and written by the background worker, so it's mutex-guarded.
type bumpyStore struct {
	mu        sync.RWMutex
	result    bumpyScoreResult
	updatedAt time.Time
	ok        bool
}

func newBumpyStore() *bumpyStore {
	return &bumpyStore{}
}

// get returns the latest result and whether one has been computed yet.
func (s *bumpyStore) get() (bumpyScoreResult, time.Time, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.result, s.updatedAt, s.ok
}

func (s *bumpyStore) set(r bumpyScoreResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.result = r
	s.updatedAt = time.Now()
	s.ok = true
}

// startMvcoBumpyWorker refreshes the BumpyScore once immediately, then on every
// tick, until ctx is cancelled. Reuses the existing MVCO fetch + parse.
func startMvcoBumpyWorker(ctx context.Context, ai AIClient, store *bumpyStore, interval time.Duration) {
	// run gates a refresh on quiet hours so we don't burn AI calls overnight.
	run := func(trigger string) {
		if inQuietHours(time.Now()) {
			log.Printf("mvco bumpy worker [%s]: quiet hours (%dpm-%dam Massachusetts time), serving quiet-hours message", trigger, quietHoursStart-12, quietHoursEnd)
			store.set(quietHoursResult())
			return
		}
		refresh(ctx, ai, store, trigger)
	}

	go func() {
		log.Printf("mvco bumpy worker: started, refreshing every %s (quiet %dpm-%dam Massachusetts time)", interval, quietHoursStart-12, quietHoursEnd)
		run("boot")

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Print("mvco bumpy worker: stopping (context cancelled)")
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
	log.Printf("mvco bumpy worker [%s]: refresh starting", trigger)

	log.Printf("mvco bumpy worker [%s]: fetching MVCO data", trigger)
	data, err := fetchMvcoData()
	if err != nil {
		log.Printf("mvco bumpy worker [%s]: fetch: %v", trigger, err)
		return
	}
	if data == nil {
		log.Printf("mvco bumpy worker [%s]: no mvco data", trigger)
		return
	}

	log.Printf("mvco bumpy worker [%s]: parsing latest reading", trigger)
	reading, err := parseMvcoRecentReading(*data)
	if err != nil {
		log.Printf("mvco bumpy worker [%s]: parse: %v", trigger, err)
		return
	}
	log.Printf("mvco bumpy worker [%s]: reading at %s, asking AI for BumpyScore", trigger, reading.Time.Format(time.RFC3339))

	result, err := calculateBumpyScoreFromMVCO(ctx, ai, reading)
	if err != nil {
		log.Printf("mvco bumpy worker [%s]: score: %v", trigger, err)
		return
	}

	store.set(result)
	if result.Score != nil {
		log.Printf("mvco bumpy worker [%s]: updated BumpyScore=%d (took %s)", trigger, *result.Score, time.Since(start).Round(time.Millisecond))
	} else {
		log.Printf("mvco bumpy worker [%s]: stored result with no score (took %s)", trigger, time.Since(start).Round(time.Millisecond))
	}
}
