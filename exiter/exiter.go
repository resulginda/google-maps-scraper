package exiter

import (
	"context"
	"sync"
	"time"
)

// Seeds bittikten sonra kalan yer sayfaları için bekleme (eski 5dk çok uzundu).
const defaultPlaceDrainGrace = 90 * time.Second

type Snapshot struct {
	SeedCount       int `json:"seed_count"`
	SeedCompleted   int `json:"seed_completed"`
	PlacesFound     int `json:"places_found"`
	PlacesCompleted int `json:"places_completed"`
}

type Exiter interface {
	SetSeedCount(int)
	SetCancelFunc(context.CancelFunc)
	IncrSeedCompleted(int)
	IncrPlacesFound(int)
	IncrPlacesCompleted(int)
	Snapshot() Snapshot
	Run(context.Context)
}

type exiter struct {
	seedCount       int
	seedCompleted   int
	placesFound     int
	placesCompleted int

	mu              *sync.Mutex
	cancelFunc      context.CancelFunc
	doneCh          chan struct{}
	seedsDoneCh     chan struct{}
	placeDrainGrace time.Duration
}

func New() Exiter {
	return &exiter{
		mu:              &sync.Mutex{},
		doneCh:          make(chan struct{}, 1),
		seedsDoneCh:     make(chan struct{}, 1),
		placeDrainGrace: defaultPlaceDrainGrace,
	}
}

func (e *exiter) SetSeedCount(val int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.seedCount = val
}

func (e *exiter) SetCancelFunc(fn context.CancelFunc) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.cancelFunc = fn
}

func (e *exiter) IncrSeedCompleted(val int) {
	e.mu.Lock()
	e.seedCompleted += val
	e.maybeSignalDoneLocked()
	e.mu.Unlock()
}

func (e *exiter) IncrPlacesFound(val int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.placesFound += val
}

func (e *exiter) IncrPlacesCompleted(val int) {
	e.mu.Lock()
	e.placesCompleted += val
	e.maybeSignalDoneLocked()
	e.mu.Unlock()
}

func (e *exiter) Snapshot() Snapshot {
	e.mu.Lock()
	defer e.mu.Unlock()

	return Snapshot{
		SeedCount:       e.seedCount,
		SeedCompleted:   e.seedCompleted,
		PlacesFound:     e.placesFound,
		PlacesCompleted: e.placesCompleted,
	}
}

func (e *exiter) maybeSignalDoneLocked() {
	if e.seedCount <= 0 {
		return
	}

	seedsDone := e.seedCompleted >= e.seedCount
	placesDone := e.placesCompleted >= e.placesFound

	switch {
	case seedsDone && placesDone:
		e.signalDoneLocked()
	case seedsDone:
		e.signalSeedsDoneLocked()
	}
}

func (e *exiter) signalDoneLocked() {
	select {
	case e.doneCh <- struct{}{}:
	default:
	}
}

func (e *exiter) signalSeedsDoneLocked() {
	select {
	case e.seedsDoneCh <- struct{}{}:
	default:
	}
}

func (e *exiter) invokeCancel() {
	if e.cancelFunc != nil {
		e.cancelFunc()
	}
}

func (e *exiter) Run(ctx context.Context) {
	var drainTimer *time.Timer
	var drainC <-chan time.Time

	defer func() {
		if drainTimer != nil {
			drainTimer.Stop()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.doneCh:
			e.invokeCancel()

			return
		case <-e.seedsDoneCh:
			if drainTimer != nil {
				drainTimer.Stop()
			}

			drainTimer = time.NewTimer(e.placeDrainGrace)
			drainC = drainTimer.C
		case <-drainC:
			e.invokeCancel()

			return
		}
	}
}
