package resilience

import (
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"
)

var ErrServiceUnavailable = errors.New("circuit breaker: service unavailable / tripped")

// CircuitBreaker implements a Google SRE style adaptive circuit breaker.
// Requests are rejected with a probability based on (requests - K * accepts) / (requests + 1).
type CircuitBreaker struct {
	mu             sync.Mutex
	k              float64
	window         time.Duration
	buckets        []bucket
	bucketDuration time.Duration
	lastTime       time.Time
}

type bucket struct {
	requests int64
	accepts  int64
}

func NewCircuitBreaker(k float64, window time.Duration, numBuckets int) *CircuitBreaker {
	if k <= 0 {
		k = 1.5 // Default Google SRE K multiplier
	}
	if window <= 0 {
		window = 10 * time.Second
	}
	if numBuckets <= 0 {
		numBuckets = 10
	}

	bDuration := window / time.Duration(numBuckets)
	return &CircuitBreaker{
		k:              k,
		window:         window,
		buckets:        make([]bucket, numBuckets),
		bucketDuration: bDuration,
		lastTime:       time.Now(),
	}
}

// Allow returns an error if the circuit breaker rejects the request.
func (cb *CircuitBreaker) Allow() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.rotate()

	var reqs, accepts int64
	for _, b := range cb.buckets {
		reqs += b.requests
		accepts += b.accepts
	}

	// Calculate rejection probability: max(0, (reqs - K * accepts) / (reqs + 1))
	dropRatio := math.Max(0, (float64(reqs)-cb.k*float64(accepts))/float64(reqs+1))
	if dropRatio > 0 && rand.Float64() < dropRatio {
		return ErrServiceUnavailable
	}

	cb.buckets[0].requests++
	return nil
}

// MarkSuccess records a successful execution.
func (cb *CircuitBreaker) MarkSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.rotate()
	cb.buckets[0].accepts++
}

// MarkFailure records a failed execution.
func (cb *CircuitBreaker) MarkFailure() {
	// Rejection already registered in requests count
}

func (cb *CircuitBreaker) rotate() {
	now := time.Now()
	elapsed := now.Sub(cb.lastTime)
	steps := int(elapsed / cb.bucketDuration)
	if steps > 0 {
		if steps >= len(cb.buckets) {
			for i := range cb.buckets {
				cb.buckets[i] = bucket{}
			}
		} else {
			copy(cb.buckets[steps:], cb.buckets)
			for i := 0; i < steps; i++ {
				cb.buckets[i] = bucket{}
			}
		}
		cb.lastTime = now
	}
}

// Do executes a fn guarded by the adaptive circuit breaker.
func (cb *CircuitBreaker) Do(fn func() error) error {
	if err := cb.Allow(); err != nil {
		return err
	}

	err := fn()
	if err != nil {
		cb.MarkFailure()
		return err
	}

	cb.MarkSuccess()
	return nil
}
