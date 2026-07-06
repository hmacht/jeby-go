package main

import (
	"context"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// MvcoRefreshInterval is how often we recompute the BumpyScore. MVCO publishes a
// new reading roughly every 20 minutes, so there's no point going faster.
const MvcoRefreshInterval = 20 * time.Minute

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
	go func() {
		log.Printf("mvco bumpy worker: started, refreshing every %s", interval)
		refresh(ctx, ai, store, "boot")

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Print("mvco bumpy worker: stopping (context cancelled)")
				return
			case <-ticker.C:
				refresh(ctx, ai, store, "tick")
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
