package resilience

import (
	"context"
	"time"
)

type RetryOptions struct {
	MaxAttempts int
	InitialWait time.Duration
	MaxWait     time.Duration
	Factor      float64
}

func DefaultRetryOptions() RetryOptions {
	return RetryOptions{
		MaxAttempts: 3,
		InitialWait: 100 * time.Millisecond,
		MaxWait:     2 * time.Second,
		Factor:      2.0,
	}
}

// DoWithRetry executes a function with exponential backoff retries.
func DoWithRetry(ctx context.Context, fn func(ctx context.Context) error, opts RetryOptions) error {
	if opts.MaxAttempts <= 0 {
		opts.MaxAttempts = 3
	}
	if opts.InitialWait <= 0 {
		opts.InitialWait = 100 * time.Millisecond
	}
	if opts.Factor <= 1.0 {
		opts.Factor = 2.0
	}

	var err error
	wait := opts.InitialWait

	for attempt := 1; attempt <= opts.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err = fn(ctx)
		if err == nil {
			return nil
		}

		if attempt == opts.MaxAttempts {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}

		wait = time.Duration(float64(wait) * opts.Factor)
		if opts.MaxWait > 0 && wait > opts.MaxWait {
			wait = opts.MaxWait
		}
	}

	return err
}
