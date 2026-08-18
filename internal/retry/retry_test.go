package retry

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// scriptedTransport plays back a fixed sequence of responses/errors and
// counts how many times RoundTrip was called.
type scriptedTransport struct {
	statuses []int   // one entry per call; 0 means "return err instead"
	errs     []error // parallel to statuses
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

func newReq(t *testing.T) *http.Request {
	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatalf("http.NewRequest: %v", err)
	}
	return req
}

func TestRetry_SucceedsFirstTry(t *testing.T) {
	mock := &scriptedTransport{statuses: []int{200}, errs: []error{nil}}
	rt := New(mock, 3, time.Millisecond)

	resp, err := rt.RoundTrip(newReq(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if mock.calls != 1 {
		t.Errorf("calls = %d, want 1", mock.calls)
	}
}

func TestRetry_RetriesOn5xxThenSucceeds(t *testing.T) {
	mock := &scriptedTransport{
		statuses: []int{500, 500, 200},
		errs:     []error{nil, nil, nil},
	}
	rt := New(mock, 3, time.Millisecond)

	resp, err := rt.RoundTrip(newReq(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if mock.calls != 3 {
		t.Errorf("calls = %d, want 3", mock.calls)
	}
}

func TestRetry_ExhaustsAttemptsOn5xx(t *testing.T) {
	mock := &scriptedTransport{
		statuses: []int{503, 503, 503},
		errs:     []error{nil, nil, nil},
	}
	rt := New(mock, 3, time.Millisecond)

	resp, err := rt.RoundTrip(newReq(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 503 {
		t.Errorf("status = %d, want 503", resp.StatusCode)
	}
	if mock.calls != 3 {
		t.Errorf("calls = %d, want 3", mock.calls)
	}
}

func TestRetry_NoRetryOn4xx(t *testing.T) {
	mock := &scriptedTransport{statuses: []int{404}, errs: []error{nil}}
	rt := New(mock, 3, time.Millisecond)

	resp, err := rt.RoundTrip(newReq(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
	if mock.calls != 1 {
		t.Errorf("calls = %d, want 1 (4xx should not be retried)", mock.calls)
	}
}

func TestRetry_RetriesOnTransportErrorThenSucceeds(t *testing.T) {
	mock := &scriptedTransport{
		statuses: []int{0, 0, 200},
		errs:     []error{errors.New("connection refused"), errors.New("connection refused"), nil},
	}
	rt := New(mock, 3, time.Millisecond)

	resp, err := rt.RoundTrip(newReq(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if mock.calls != 3 {
		t.Errorf("calls = %d, want 3", mock.calls)
	}
}

func TestRetry_ReturnsErrorAfterExhaustingAttempts(t *testing.T) {
	wantErr := errors.New("connection refused")
	mock := &scriptedTransport{
		statuses: []int{0, 0, 0},
		errs:     []error{wantErr, wantErr, wantErr},
	}
	rt := New(mock, 3, time.Millisecond)

	resp, err := rt.RoundTrip(newReq(t))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if resp != nil {
		t.Errorf("resp = %v, want nil", resp)
	}
	if mock.calls != 3 {
		t.Errorf("calls = %d, want 3", mock.calls)
	}
}
