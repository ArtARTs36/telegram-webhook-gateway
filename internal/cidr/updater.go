package cidr

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultCIDRUpdateTimeout = 10 * time.Second
)

// PeriodicUpdater performs periodic CIDR updates in the Store.
type PeriodicUpdater struct {
	store    *Store
	fetcher  Fetcher
	interval time.Duration
	timeout  time.Duration

	updating atomic.Bool
	stopping chan struct{}
	wg       sync.WaitGroup
}

// NewPeriodicUpdater creates a new PeriodicUpdater.
func NewPeriodicUpdater(store *Store, fetcher Fetcher, interval time.Duration) *PeriodicUpdater {
	return &PeriodicUpdater{
		store:    store,
		fetcher:  fetcher,
		interval: interval,
		timeout:  defaultCIDRUpdateTimeout,
		stopping: make(chan struct{}, 1),
	}
}

// Run launches the periodic CIDR update loop.
// The first load is performed synchronously so the gateway starts with up-to-date data.
func (u *PeriodicUpdater) Run() {
	u.wg = sync.WaitGroup{}
	u.wg.Add(1)

	ticker := time.NewTicker(u.interval)

	slog.Info("[cidr] updater started", slog.String("interval", u.interval.String()))

	ctx, cancel := context.WithCancel(context.Background())

	tryUpdate := func() {
		if u.updating.CompareAndSwap(false, true) {
			return
		}

		if err := u.Update(ctx); err != nil {
			slog.WarnContext(ctx, "[cidr] failed to update cidr list", slog.Any("err", err))
		}

		u.updating.Swap(false)
	}

	for {
		select {
		case <-u.stopping:
			cancel()
			slog.Info("[cidr] updater stopped")
			u.wg.Done()
		case <-ticker.C:
			tryUpdate()
		}
	}
}

func (u *PeriodicUpdater) Stop() {
	u.stopping <- struct{}{}
	u.wg.Wait()
}

func (u *PeriodicUpdater) Update(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, u.timeout)
	defer cancel()

	nets, err := u.fetcher.Fetch(ctx)
	if err != nil {
		return fmt.Errorf("fetch list: %w", err)
	}

	if len(nets) == 0 {
		slog.Default().WarnContext(ctx, "CIDR update returned empty list, keeping previous CIDRs")
		return nil
	}

	u.store.Set(nets)
	slog.Default().InfoContext(ctx, "updated Telegram CIDRs", "count", len(nets))

	return nil
}
