package breaker

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// scriptedTransport plays back a fixed sequence of responses/errors and
// counts how many times RoundTrip was called. Same pattern as
// internal/retry/retry_test.go's mock.
type scriptedTransport struct {
	statuses []int
	errs     []error
	calls    int
}

func (s *scriptedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	i := s.calls
	s.calls++
	if s.errs[i] != nil {
		return nil, s.errs[i]
	}
	return &http.Response{
		StatusCode: s.statuses[i],
		Body:       io.NopCloser(strings.NewReader("")),
	}, nil
}

// blockingTransport lets a test pause a RoundTrip mid-flight, so it can
// assert on state (like probeInFlight) while the "network call" is still
// outstanding.
type blockingTransport struct {
	started chan struct{}
	release chan struct{}
	calls   int32
}

func (b *blockingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	atomic.AddInt32(&b.calls, 1)
	close(b.started)
	<-b.release
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))}, nil
}

func newReq(t *testing.T) *http.Request {
	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatalf("http.NewRequest: %v", err)
	}
	return req
}

// forceState reaches into the unexported fields to set up a starting state
// directly, instead of sleeping in real time to make a cooldown "elapse".
// Valid since this test file lives in package breaker, not breaker_test.
func forceState(cbt *CircuitBreakerTransport, state State, tripOpenTime time.Time) {
	cbt.mu.Lock()
	defer cbt.mu.Unlock()
	cbt.state = state
	cbt.tripOpenTime = tripOpenTime
}

func TestClosed_AllowsAndForwardsSuccess(t *testing.T) {
	mock := &scriptedTransport{statuses: []int{200}, errs: []error{nil}}
	cbt := New(mock, 3, time.Minute)

	resp, err := cbt.RoundTrip(newReq(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if mock.calls != 1 {
		t.Errorf("calls = %d, want 1", mock.calls)
	}
	if cbt.state != closed {
		t.Errorf("state = %v, want closed", cbt.state)
	}
}

func TestClosed_DoesNotTripBeforeThreshold(t *testing.T) {
	mock := &scriptedTransport{
		statuses: []int{500, 500},
		errs:     []error{nil, nil},
	}
	cbt := New(mock, 3, time.Minute)

	for i := 0; i < 2; i++ {
		_, err := cbt.RoundTrip(newReq(t))
		if err != nil {
			t.Fatalf("call %d: unexpected error: %v", i+1, err)
		}
	}
	if cbt.state != closed {
		t.Errorf("state = %v, want closed (threshold not reached)", cbt.state)
	}
	if cbt.failureCount != 2 {
		t.Errorf("failureCount = %d, want 2", cbt.failureCount)
	}
	if mock.calls != 2 {
		t.Errorf("calls = %d, want 2", mock.calls)
	}
}

func TestClosed_TripsOpenAtThreshold(t *testing.T) {
	mock := &scriptedTransport{
		statuses: []int{500, 500, 500},
		errs:     []error{nil, nil, nil},
	}
	cbt := New(mock, 3, time.Minute)

	for i := 0; i < 3; i++ {
		cbt.RoundTrip(newReq(t))
	}
	if cbt.state != open {
		t.Errorf("state = %v, want open after %d consecutive failures", cbt.state, cbt.failureCount)
	}
	if mock.calls != 3 {
		t.Errorf("calls = %d, want 3", mock.calls)
	}
}

func TestOpen_RejectsWithoutCallingTransport(t *testing.T) {
	mock := &scriptedTransport{statuses: []int{}, errs: []error{}}
	cbt := New(mock, 1, time.Hour)
	forceState(cbt, open, time.Now()) // just tripped, cooldown far from elapsed

	resp, err := cbt.RoundTrip(newReq(t))
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("err = %v, want ErrCircuitOpen", err)
	}
	if resp != nil {
		t.Errorf("resp = %v, want nil", resp)
	}
	if mock.calls != 0 {
		t.Errorf("calls = %d, want 0 (transport must not be touched while open)", mock.calls)
	}
}

func TestOpen_AfterCooldownProbeSucceeds_Closes(t *testing.T) {
	mock := &scriptedTransport{statuses: []int{200}, errs: []error{nil}}
	cbt := New(mock, 1, 10*time.Millisecond)
	forceState(cbt, open, time.Now().Add(-time.Hour)) // cooldown long elapsed

	resp, err := cbt.RoundTrip(newReq(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if mock.calls != 1 {
		t.Errorf("calls = %d, want 1 (probe should reach the transport)", mock.calls)
	}
	if cbt.state != closed {
		t.Errorf("state = %v, want closed after successful probe", cbt.state)
	}
	if cbt.probeInFlight {
		t.Error("probeInFlight = true, want false after probe resolved")
	}
}

func TestOpen_AfterCooldownProbeFails_ReopensAndResetsCooldown(t *testing.T) {
	mock := &scriptedTransport{statuses: []int{500}, errs: []error{nil}}
	cbt := New(mock, 1, 10*time.Millisecond)
	forceState(cbt, open, time.Now().Add(-time.Hour)) // cooldown long elapsed

	cbt.RoundTrip(newReq(t))

	if cbt.state != open {
		t.Errorf("state = %v, want open after failed probe", cbt.state)
	}
	if cbt.probeInFlight {
		t.Error("probeInFlight = true, want false after probe resolved")
	}
	if mock.calls != 1 {
		t.Errorf("calls = %d, want 1", mock.calls)
	}
	// tripOpenTime must have been reset to "now", not left at the old,
	// already-elapsed timestamp - otherwise the very next call would
	// wrongly treat the cooldown as elapsed again immediately.
	if time.Since(cbt.tripOpenTime) > time.Second {
		t.Errorf("tripOpenTime not reset on reopen: %v ago", time.Since(cbt.tripOpenTime))
	}
}

func TestHalfOpen_OnlyOneProbeAllowedConcurrently(t *testing.T) {
	bt := &blockingTransport{started: make(chan struct{}), release: make(chan struct{})}
	cbt := New(bt, 1, 10*time.Millisecond)
	forceState(cbt, open, time.Now().Add(-time.Hour)) // cooldown long elapsed

	firstErr := make(chan error, 1)
	go func() {
		_, err := cbt.RoundTrip(newReq(t))
		firstErr <- err
	}()

	<-bt.started // first call is now the probe, mid-flight, holding probeInFlight=true

	_, err := cbt.RoundTrip(newReq(t))
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("second concurrent call: err = %v, want ErrCircuitOpen", err)
	}

	close(bt.release) // let the first (probe) call finish
	if err := <-firstErr; err != nil {
		t.Errorf("first (probe) call: unexpected error: %v", err)
	}

	if calls := atomic.LoadInt32(&bt.calls); calls != 1 {
		t.Errorf("transport calls = %d, want 1 (only the probe should reach it)", calls)
	}
}
