package exiter

import (
	"context"
	"testing"
	"time"
)

func TestExiterSignalsDoneWhenSeedsAndPlacesComplete(t *testing.T) {
	t.Parallel()

	m := New().(*exiter)
	m.SetSeedCount(2)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	canceled := make(chan struct{})
	m.SetCancelFunc(func() {
		close(canceled)
	})

	go m.Run(ctx)

	m.IncrSeedCompleted(1)
	m.IncrPlacesFound(1)
	m.IncrPlacesCompleted(1)
	m.IncrSeedCompleted(1)

	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("expected cancel when seeds and places complete")
	}
}

func TestExiterForcesCancelAfterSeedDrainGrace(t *testing.T) {
	t.Parallel()

	m := New().(*exiter)
	m.placeDrainGrace = 200 * time.Millisecond
	m.SetSeedCount(1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	canceled := make(chan struct{})
	m.SetCancelFunc(func() {
		close(canceled)
	})

	go m.Run(ctx)

	m.IncrSeedCompleted(1)
	m.IncrPlacesFound(3)
	m.IncrPlacesCompleted(1)

	select {
	case <-canceled:
	case <-time.After(2 * time.Second):
		t.Fatal("expected cancel after place drain grace when places remain")
	}
}

func TestExiterDoesNotSignalBeforeSeedCountSet(t *testing.T) {
	t.Parallel()

	m := New().(*exiter)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	canceled := make(chan struct{})
	m.SetCancelFunc(func() {
		close(canceled)
	})

	go m.Run(ctx)

	m.IncrSeedCompleted(1)

	select {
	case <-canceled:
		t.Fatal("did not expect cancel before seed count is configured")
	case <-time.After(50 * time.Millisecond):
	}
}
