package breaker

import (
	"errors"
	"net/http"
	"sync"
	"time"
)

// ErrCircuitOpen is returned instead of calling the underlying transport
// when the breaker is open (or half-open with a probe already in flight).
var ErrCircuitOpen = errors.New("breaker: circuit open")

type State int

const (
	closed State = iota
	open
	halfOpen
)

type CircuitBreakerTransport struct {
	wrap          http.RoundTripper
	state         State
	failureCount  int
	threshold     int
	tripOpenTime  time.Time
	probeInFlight bool
	mu            sync.Mutex
	cooldown      time.Duration
}

func New(transport http.RoundTripper, threshold int, cooldown time.Duration) *CircuitBreakerTransport {
	return &CircuitBreakerTransport{wrap: transport, threshold: threshold, cooldown: cooldown}
}

func (cbt *CircuitBreakerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	allowed, isProbe := cbt.before()
	if !allowed {
		return nil, ErrCircuitOpen
	}

	resp, err := cbt.wrap.RoundTrip(req)
	failed := err != nil || (resp != nil && resp.StatusCode >= 500)

	cbt.after(isProbe, failed)

	return resp, err
}

// before decides, under lock, whether this call may proceed and whether it
// is acting as the half-open probe. It performs the closed->open->half-open
// transition itself since "cooldown elapsed" only gets checked once, here.
func (cbt *CircuitBreakerTransport) before() (allowed bool, isProbe bool) {
	cbt.mu.Lock()
	defer cbt.mu.Unlock()

	switch cbt.state {
	case closed:
		return true, false

	case open:
		if time.Since(cbt.tripOpenTime) < cbt.cooldown {
			return false, false
		}
		// cooldown elapsed: this call becomes the probe
		cbt.state = halfOpen
		cbt.probeInFlight = true
		return true, true

	case halfOpen:
		if cbt.probeInFlight {
			return false, false
		}
		cbt.probeInFlight = true
		return true, true

	default:
		return true, false
	}
}

// after records the outcome of a call that was allowed to proceed, and
// applies the resulting state transition.
func (cbt *CircuitBreakerTransport) after(isProbe bool, failed bool) {
	cbt.mu.Lock()
	defer cbt.mu.Unlock()

	if isProbe {
		cbt.probeInFlight = false
		if failed {
			cbt.state = open
			cbt.tripOpenTime = time.Now()
		} else {
			cbt.state = closed
			cbt.failureCount = 0
		}
		return
	}

	if failed {
		cbt.failureCount++
		if cbt.failureCount >= cbt.threshold {
			cbt.state = open
			cbt.tripOpenTime = time.Now()
		}
	} else {
		cbt.failureCount = 0
	}
}
