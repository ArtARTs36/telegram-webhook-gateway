package cidr

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const (
	defaultCIDRUpdateTimeout = 10 * time.Second
)

// PeriodicUpdater performs periodic CIDR updates in the Store.
type PeriodicUpdater struct {
	store      *Store
	fetcher    Fetcher
	interval   time.Duration
	timeout    time.Duration
	once       sync.Once
	updatedCh  chan struct{}
	updateDone chan struct{}
}

// NewPeriodicUpdater creates a new PeriodicUpdater.
func NewPeriodicUpdater(store *Store, fetcher Fetcher, interval time.Duration) *PeriodicUpdater {
	return &PeriodicUpdater{
		store:      store,
		fetcher:    fetcher,
		interval:   interval,
		timeout:    defaultCIDRUpdateTimeout,
		updatedCh:  make(chan struct{}, 1),
		updateDone: make(chan struct{}),
	}
}

// Start launches the periodic CIDR update loop.
// The first load is performed synchronously so the gateway starts with up-to-date data.
func (u *PeriodicUpdater) Start() {
	u.once.Do(func() {
		// initial load
		if err := u.updateOnce(); err != nil {
			slog.Default().ErrorContext(context.Background(), "initial CIDR update failed", "err", err)
		}

		ticker := time.NewTicker(u.interval)
		go func() {
			defer close(u.updateDone)
			for range ticker.C {
				if err := u.updateOnce(); err != nil {
					slog.Default().ErrorContext(context.Background(), "periodic CIDR update failed", "err", err)
				}
			}
		}()
	})
}

func (u *PeriodicUpdater) updateOnce() error {
	ctx, cancel := context.WithTimeout(context.Background(), u.timeout)
	defer cancel()

	nets, err := u.fetcher.Fetch(ctx)
	if err != nil {
		return err
	}

	if len(nets) == 0 {
		slog.Default().WarnContext(ctx, "CIDR update returned empty list, keeping previous CIDRs")
		return nil
	}

	u.store.Set(nets)
	slog.Default().InfoContext(ctx, "updated Telegram CIDRs", "count", len(nets))

	select {
	case u.updatedCh <- struct{}{}:
	default:
	}

	return nil
}
