package app

import (
	"context"
	"sync"
	"testing"
	"time"

	"macstats/internal/collector"
)

type fakeWriter struct {
	mu     sync.Mutex
	writes [][]byte
}

func (w *fakeWriter) WriteFrame(buffer []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	copyBuf := append([]byte(nil), buffer...)
	w.writes = append(w.writes, copyBuf)
	return nil
}

func (w *fakeWriter) count() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.writes)
}

type fakeCollector struct {
	mu    sync.Mutex
	calls int
}

func (c *fakeCollector) CollectMetrics() (collector.Metrics, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.calls++
	return collector.Metrics{}, nil
}

func (c *fakeCollector) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

type fakeFrameSource struct {
	startup []byte
	refresh []byte
}

func (s fakeFrameSource) StartupPayload() []byte {
	return append([]byte(nil), s.startup...)
}

func (s fakeFrameSource) RefreshPayload(collector.Metrics) []byte {
	return append([]byte(nil), s.refresh...)
}

type manualTicker struct {
	ch chan time.Time
}

func (t *manualTicker) C() <-chan time.Time {
	return t.ch
}

func (t *manualTicker) Stop() {}

func TestRunnerWritesStartupAndRefreshPayloads(t *testing.T) {
	writer := &fakeWriter{}
	collector := &fakeCollector{}
	ticker := &manualTicker{ch: make(chan time.Time, 1)}
	runner := &Runner{
		Writer:          writer,
		Collector:       collector,
		Frames:          fakeFrameSource{startup: []byte{0x01, 0x02, 0x03}, refresh: []byte{0x04, 0x05}},
		RefreshInterval: time.Second,
		NewTicker: func(time.Duration) Ticker {
			return ticker
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runner.Run(ctx)
	}()

	time.Sleep(10 * time.Millisecond)
	if got := writer.count(); got != 1 {
		t.Fatalf("expected one startup write, got %d", got)
	}

	ticker.ch <- time.Now()
	time.Sleep(10 * time.Millisecond)
	if got := writer.count(); got != 2 {
		t.Fatalf("expected one refresh write, got %d total writes", got)
	}
	if got := collector.count(); got != 1 {
		t.Fatalf("expected one collector call, got %d", got)
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected run error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("runner did not exit after cancellation")
	}
}
