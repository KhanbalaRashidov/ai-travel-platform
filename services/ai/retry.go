package ai

import (
	"context"
	"math/rand"
	"time"
)

// retryTransient runs do() up to len(delays)+1 times, retrying ONLY transient
// provider errors (429/503) with the given backoff waits plus up to 25% jitter.
// 4xx and parse errors return immediately; it stops early if ctx is cancelled.
// Shared by the Gemini and Groq adapters.
func retryTransient(ctx context.Context, delays []time.Duration, sleep func(time.Duration), do func() (Completion, error)) (Completion, error) {
	var last error
	for i := 0; i <= len(delays); i++ {
		if i > 0 {
			if d := delays[i-1]; d > 0 {
				sleep(d + jitter(d))
			}
			if err := ctx.Err(); err != nil {
				return Completion{}, err
			}
		}
		comp, err := do()
		if err == nil {
			return comp, nil
		}
		last = err
		if !IsTransient(err) {
			return Completion{}, err
		}
	}
	return Completion{}, last // retries exhausted; last is a transient *APIError
}

func jitter(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}
	return time.Duration(rand.Int63n(int64(d)/4 + 1))
}
