package fx

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/KhanbalaRashidov/ai-travel-platform/services/connectors"
)

func TestParseRate_OK(t *testing.T) {
	body := []byte(`{"base":"USD","date":"2024-06-01","rates":{"EUR":0.92}}`)
	at := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	r, err := parseRate(body, "USD", "EUR", at, "frankfurter-ecb")
	if err != nil {
		t.Fatal(err)
	}
	if r.Rate != 0.92 || r.Base != "USD" || r.Quote != "EUR" {
		t.Fatalf("bad parse: %+v", r)
	}
	if r.Source != "frankfurter-ecb" || !r.ExpiresAt.Equal(endOfDay(at)) {
		t.Fatalf("bad provenance: %+v", r.Provenance)
	}
}

func TestParseRate_MissingQuote(t *testing.T) {
	body := []byte(`{"base":"USD","date":"2024-06-01","rates":{"GBP":0.79}}`)
	_, err := parseRate(body, "USD", "EUR", time.Now(), "frankfurter-ecb")
	if !errors.Is(err, connectors.ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable, got %v", err)
	}
}

func TestRate_SameCurrency(t *testing.T) {
	f := New(nil)
	r, err := f.Rate(context.Background(), "usd", "USD")
	if err != nil || r.Rate != 1 {
		t.Fatalf("same currency should be 1.0, got %v (%v)", r.Rate, err)
	}
}

type fakeDoer struct {
	resp *http.Response
	err  error
}

func (d fakeDoer) Do(*http.Request) (*http.Response, error) { return d.resp, d.err }

func TestRate_WithFakeHTTP(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"base":"EUR","date":"2024-06-01","rates":{"AZN":1.84}}`)),
	}
	f := New(fakeDoer{resp: resp})
	r, err := f.Rate(context.Background(), "EUR", "AZN")
	if err != nil {
		t.Fatal(err)
	}
	if r.Rate != 1.84 || r.Quote != "AZN" {
		t.Fatalf("unexpected rate: %+v", r)
	}
}
