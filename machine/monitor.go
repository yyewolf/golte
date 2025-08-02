package machine

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"golte/config"
)

// SignalMonitor handles signal quality monitoring
type SignalMonitor struct {
	config      *config.Config
	logger      *slog.Logger
	modem       *ModemManager
	ctx         context.Context
	cancel      context.CancelFunc
	wg          *sync.WaitGroup
	stopChannel chan struct{}
}

// NewSignalMonitor creates a new SignalMonitor instance
func NewSignalMonitor(cfg *config.Config, modem *ModemManager, wg *sync.WaitGroup) *SignalMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	return &SignalMonitor{
		config:      cfg,
		logger:      slog.With("component", "signal-monitor"),
		modem:       modem,
		ctx:         ctx,
		cancel:      cancel,
		wg:          wg,
		stopChannel: make(chan struct{}),
	}
}

// Start begins signal quality monitoring
func (s *SignalMonitor) Start() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		s.logger.Info("Starting signal quality monitoring")

		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				result, err := s.modem.GetSignalQuality()
				if err != nil {
					s.logger.Error("Failed to get signal quality", slog.Any("error", err))
				} else {
					s.logger.Debug("Signal quality", slog.Any("result", result))
				}
			case <-s.ctx.Done():
				s.logger.Info("Signal quality monitoring stopped")
				return
			case <-s.modem.Closed():
				s.logger.Warn("Modem closed, stopping signal quality monitoring")
				return
			case <-s.stopChannel:
				s.logger.Info("Signal quality monitoring stopped via stop channel")
				return
			}
		}
	}()
}

// Stop stops signal quality monitoring
func (s *SignalMonitor) Stop() {
	s.cancel()
	close(s.stopChannel)
}

// SetContext updates the context for cancellation
func (s *SignalMonitor) SetContext(ctx context.Context) {
	s.cancel() // Cancel the old context
	s.ctx = ctx
}
