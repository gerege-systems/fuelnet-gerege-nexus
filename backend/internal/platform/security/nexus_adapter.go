/*
 * Gerege Nexus
 * Copyright (c) 2026 Gerege Systems Development Team, Gerege Nomadica Foundation
 * Distributed under the Apache 2.0 License.
 */

package security

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"

	"github.com/gerege-systems/open-gerege-nexus/backend/pkg/nexus"
)

// AsRateLimiter presents this package's limiters as nexus.RateLimiter.
//
// One limiter per (perMinute, burst) pair, built on demand and kept: a module
// asking for the same budget twice must get the same bucket, or two routes
// meant to share a limit would each get a full one.
//
// Both halves, not just the local one. The SDK has promised a module "a rate
// limiter whose budget is shared across the deployment" since the capability
// existed, and until 2026-08-23 it handed back a per-replica bucket: three
// replicas behind one gateway enforced three budgets and the deployment had
// none. The platform's own expensive routes — sign-in, polling, the assistant
// — were paired with a Redis counter by hand in server.go, which is exactly the
// thing a module cannot do for itself. client may be nil, and a nil shared
// limiter allows: a deployment without Redis is back to the local bucket, which
// is what it had before.
func AsRateLimiter(client *redis.Client) nexus.RateLimiter {
	return &limiterFactory{client: client, made: map[budget]func(http.Handler) http.Handler{}}
}

type budget struct {
	perMinute float64
	burst     int
}

type limiterFactory struct {
	client *redis.Client
	mu     sync.Mutex
	made   map[budget]func(http.Handler) http.Handler
}

func (f *limiterFactory) Limit(perMinute float64, burst int) func(http.Handler) http.Handler {
	key := budget{perMinute, burst}
	f.mu.Lock()
	defer f.mu.Unlock()
	if made, ok := f.made[key]; ok {
		return made
	}
	local := NewIPRateLimiter(rate.Limit(perMinute/60.0), burst)
	// The Redis key is the budget itself, so two modules asking for the same
	// numbers share one deployment-wide counter and two asking for different
	// ones do not — the same rule the local map follows one line up.
	shared := NewSharedLimiter(f.client, fmt.Sprintf("module-%g-%d", perMinute, burst), int(perMinute), time.Minute)
	made := SharedRateLimitMiddleware(local, shared)
	f.made[key] = made
	return made
}
