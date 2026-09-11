package health_test

import (
	"context"
	"testing"
	"time"

	gosundheit "github.com/AppsFlyer/go-sundheit"
	"github.com/AppsFlyer/go-sundheit/checks"

	"github.com/dexidp/dex/pkg/health"
)

// Exercises the wrapper through the health scheduler itself: a check that stops
// returning must still flip the scheduler unhealthy, rather than leaving the
// last recorded result standing. See health.NewTimeoutCheckFunc.
func TestHealthCheckerReportsUnresponsiveDependency(t *testing.T) {
	stall := make(chan struct{})
	defer close(stall)

	healthy := make(chan struct{})
	blocked := make(chan struct{})

	// Healthy until blocked is closed, then never returns, ignoring context.
	check := func(ctx context.Context) (interface{}, error) {
		select {
		case <-blocked:
			<-stall
			return nil, nil
		default:
			select {
			case healthy <- struct{}{}:
			default:
			}
			return nil, nil
		}
	}

	h := gosundheit.New()
	err := h.RegisterCheck(
		&checks.CustomCheck{
			CheckName: "storage",
			CheckFunc: health.NewTimeoutCheckFunc(check, 50*time.Millisecond),
		},
		gosundheit.ExecutionPeriod(100*time.Millisecond),
		gosundheit.InitiallyPassing(true),
	)
	if err != nil {
		t.Fatalf("register check: %v", err)
	}
	defer h.DeregisterAll()

	// Establish healthy first, so the assertion below is a transition.
	select {
	case <-healthy:
	case <-time.After(5 * time.Second):
		t.Fatal("check never ran while healthy")
	}
	if !h.IsHealthy() {
		t.Fatal("expected healthy before the dependency becomes unresponsive")
	}

	close(blocked)

	deadline := time.Now().Add(10 * time.Second)
	for {
		if !h.IsHealthy() {
			return // reported unhealthy
		}
		if time.Now().After(deadline) {
			t.Fatal("health checker still reports healthy after the dependency " +
				"stopped responding: a check that never returns must not leave " +
				"the last passing result standing")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
