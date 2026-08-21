/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

package security

import (
	"net/http"
	"sync"

	"golang.org/x/time/rate"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
)

// AsRateLimiter presents this package's per-IP limiter as nexus.RateLimiter.
//
// One limiter per (perMinute, burst) pair, built on demand and kept: a module
// asking for the same budget twice must get the same bucket, or two routes
// meant to share a limit would each get a full one.
func AsRateLimiter() nexus.RateLimiter { return &limiterFactory{made: map[budget]*IPRateLimiter{}} }

type budget struct {
	perMinute float64
	burst     int
}

type limiterFactory struct {
	mu   sync.Mutex
	made map[budget]*IPRateLimiter
}

func (f *limiterFactory) Limit(perMinute float64, burst int) func(http.Handler) http.Handler {
	key := budget{perMinute, burst}
	f.mu.Lock()
	limiter, ok := f.made[key]
	if !ok {
		limiter = NewIPRateLimiter(rate.Limit(perMinute/60.0), burst)
		f.made[key] = limiter
	}
	f.mu.Unlock()
	return RateLimitMiddleware(limiter)
}
