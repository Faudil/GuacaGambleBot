package watchdog

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
)

// fakePinger lets tests simulate DB health without a real driver.
type fakePinger struct {
	err error
}

func (f *fakePinger) PingContext(ctx context.Context) error { return f.err }

// seqPinger returns a fixed sequence of results, then succeeds forever. It
// makes transient-failure tests deterministic.
type seqPinger struct {
	mu   sync.Mutex
	errs []error
}

func (p *seqPinger) PingContext(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.errs) == 0 {
		return nil
	}
	e := p.errs[0]
	p.errs = p.errs[1:]
	return e
}

func TestGatewayHealthy(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"success", nil, true},
		{"rate limited (429)", &discordgo.RateLimitError{}, true},
		{"http error response", &discordgo.RESTError{}, true},
		{"wrapped rate limit", errors.Join(errors.New("wrapped"), &discordgo.RateLimitError{}), true},
		{"connection refused", errors.New("dial tcp: connection refused"), false},
		{"context deadline", context.DeadlineExceeded, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := gatewayHealthy(c.err); got != c.want {
				t.Fatalf("gatewayHealthy(%v) = %v, want %v", c.err, got, c.want)
			}
		})
	}
}

func TestCheckHealthyTouchesHeartbeat(t *testing.T) {
	dir := t.TempDir()
	hb := filepath.Join(dir, "heartbeat")

	w := New(&fakePinger{}, nil, Options{
		Interval:      5 * time.Millisecond,
		MaxFailures:   1,
		HeartbeatPath: hb,
	})

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); w.Run(ctx) }()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(hb); err == nil {
			cancel()
			wg.Wait()
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	wg.Wait()
	t.Fatal("heartbeat file was never written on healthy checks")
}

func TestCheckDBFailureExitsAfterMaxFailures(t *testing.T) {
	dir := t.TempDir()
	hb := filepath.Join(dir, "heartbeat")

	var fatalCalls []string
	w := New(&fakePinger{err: errors.New("boom")}, nil, Options{
		Interval:      5 * time.Millisecond,
		MaxFailures:   2,
		HeartbeatPath: hb,
	})
	w.fatal = func(reason string) { fatalCalls = append(fatalCalls, reason) }

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w.Run(ctx) // Run returns after fatal fires (MaxFailures reached)

	if len(fatalCalls) != 1 {
		t.Fatalf("expected 1 fatal call, got %d (%v)", len(fatalCalls), fatalCalls)
	}
	if _, err := os.Stat(hb); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("heartbeat file must not be written on unhealthy checks")
	}
}

func TestCheckRecoversAfterTransientFailure(t *testing.T) {
	dir := t.TempDir()
	hb := filepath.Join(dir, "heartbeat")

	var fatalCalls []string
	w := New(&seqPinger{errs: []error{errors.New("boom"), errors.New("boom")}}, nil, Options{
		Interval:      5 * time.Millisecond,
		MaxFailures:   3,
		HeartbeatPath: hb,
	})
	w.fatal = func(reason string) { fatalCalls = append(fatalCalls, reason) }

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); w.Run(ctx) }()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(hb); err == nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	wg.Wait()

	if len(fatalCalls) != 0 {
		t.Fatalf("transient failure must not trigger exit, got %v", fatalCalls)
	}
	if _, err := os.Stat(hb); err != nil {
		t.Fatal("heartbeat file was not written after recovery")
	}
}
