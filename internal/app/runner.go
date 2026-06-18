package app

import (
	"context"
	"time"

	"macstats/internal/collector"
)

type Writer interface {
	WriteFrame([]byte) error
}

type Collector interface {
	CollectMetrics() (collector.Metrics, error)
}

type FrameSource interface {
	StartupPayload() []byte
	RefreshPayload(collector.Metrics) []byte
}

type Ticker interface {
	C() <-chan time.Time
	Stop()
}

type tickerShim struct {
	ticker *time.Ticker
}

func (t *tickerShim) C() <-chan time.Time {
	return t.ticker.C
}

func (t *tickerShim) Stop() {
	t.ticker.Stop()
}

type Runner struct {
	Writer          Writer
	Collector       Collector
	Frames          FrameSource
	RefreshInterval time.Duration
	NewTicker       func(time.Duration) Ticker
}

func (r *Runner) Run(ctx context.Context) error {
	if r.Frames == nil {
		return nil
	}

	if err := r.Writer.WriteFrame(r.Frames.StartupPayload()); err != nil {
		return err
	}

	newTicker := r.NewTicker
	if newTicker == nil {
		newTicker = func(interval time.Duration) Ticker {
			return &tickerShim{ticker: time.NewTicker(interval)}
		}
	}

	interval := r.RefreshInterval
	if interval <= 0 {
		interval = time.Second
	}

	ticker := newTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C():
			metrics := collector.Metrics{}
			if r.Collector != nil {
				collected, err := r.Collector.CollectMetrics()
				if err == nil {
					metrics = collected
				}
			}
			if err := r.Writer.WriteFrame(r.Frames.RefreshPayload(metrics)); err != nil {
				return err
			}
		}
	}
}
