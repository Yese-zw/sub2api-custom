package service

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type UpstreamBalanceRefreshService struct {
	upstreamBalanceService *UpstreamBalanceService
	interval               time.Duration
	stopCh                 chan struct{}
	stopOnce               sync.Once
	wg                     sync.WaitGroup
}

func NewUpstreamBalanceRefreshService(upstreamBalanceService *UpstreamBalanceService, interval time.Duration) *UpstreamBalanceRefreshService {
	return &UpstreamBalanceRefreshService{
		upstreamBalanceService: upstreamBalanceService,
		interval:               interval,
		stopCh:                 make(chan struct{}),
	}
}

func (s *UpstreamBalanceRefreshService) Start() {
	if s == nil || s.upstreamBalanceService == nil || s.interval <= 0 {
		return
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		s.runOnce()
		for {
			select {
			case <-ticker.C:
				s.runOnce()
			case <-s.stopCh:
				return
			}
		}
	}()
}

func (s *UpstreamBalanceRefreshService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

func (s *UpstreamBalanceRefreshService) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	result, err := s.upstreamBalanceService.RefreshAccounts(ctx, UpstreamBalanceRefreshOptions{
		SkipRecentlyFresh: false,
		NotifyLowBalance:  true,
	}, nil)
	if err != nil {
		slog.Warn("upstream_balance_scheduled_refresh_failed", "error", err)
		return
	}
	if result.Success > 0 || result.Failed > 0 || result.Skipped > 0 {
		slog.Info("upstream_balance_scheduled_refresh_finished",
			"success", result.Success,
			"failed", result.Failed,
			"skipped", result.Skipped,
		)
	}
}
