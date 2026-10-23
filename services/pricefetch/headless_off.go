//go:build !headless

package pricefetch

// NewHeadless returns nil in the default build: no Chromium/Playwright is linked,
// so `go build`/`go test` and the standard Docker image stay lean. Build with
// `-tags headless` (and the playwright-go dependency + Chromium in the image) to
// enable JS rendering — see headless_playwright.go.
func NewHeadless() Renderer { return nil }
