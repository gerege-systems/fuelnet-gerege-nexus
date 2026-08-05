package resilience

import (
	"net/http"
	"sync/atomic"
)

type LoadShedder struct {
	maxInFlight int64
	current     int64
}

func NewLoadShedder(maxInFlight int64) *LoadShedder {
	if maxInFlight <= 0 {
		maxInFlight = 1000 // Default max concurrent requests limit
	}
	return &LoadShedder{
		maxInFlight: maxInFlight,
	}
}

// Middleware returns HTTP middleware that sheds load when system capacity is reached.
func (ls *LoadShedder) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt64(&ls.current, 1)
		defer atomic.AddInt64(&ls.current, -1)

		if current > ls.maxInFlight {
			w.Header().Set("Retry-After", "5")
			http.Error(w, `{"error":"service overloaded: load shedding active"}`, http.StatusServiceUnavailable)
			return
		}

		next.ServeHTTP(w, r)
	})
}
