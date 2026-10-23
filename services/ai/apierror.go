package ai

import (
	"errors"
	"fmt"
)

// APIError is a non-2xx response from an LLM provider. Transient() marks the
// errors worth retrying (the provider is momentarily overloaded/rate-limited),
// as opposed to client errors (400/401/404) or parse failures, which are not.
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string { return fmt.Sprintf("ai http %d: %s", e.Status, e.Message) }

// Transient reports whether the status is worth a retry (429 or 503).
func (e *APIError) Transient() bool { return e.Status == 429 || e.Status == 503 }

// IsTransient reports whether err is a transient provider error (429/503).
func IsTransient(err error) bool {
	var ae *APIError
	return errors.As(err, &ae) && ae.Transient()
}
