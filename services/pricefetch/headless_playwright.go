//go:build headless

// Headless (Playwright/Chromium) renderer. Compiled only with `-tags headless`.
// Enable it with:
//
//	go get github.com/playwright-community/playwright-go
//	# in the image: install Chromium + its deps
//	go run github.com/playwright-community/playwright-go/cmd/playwright@latest install --with-deps chromium
//	go build -tags headless ./services/gateway
//
// It launches Chromium headless with container-friendly flags (--no-sandbox) and
// returns the fully rendered DOM text, for JS-rendered price pages. It never
// solves CAPTCHAs or rotates identity — a blocked page simply yields its (empty)
// content and the source is skipped upstream.
package pricefetch

import (
	"context"
	"time"

	"github.com/playwright-community/playwright-go"
)

type playwrightRenderer struct {
	pw *playwright.Playwright
}

// NewHeadless starts Playwright once and returns a headless renderer.
func NewHeadless() Renderer {
	pw, err := playwright.Run()
	if err != nil {
		return nil // Playwright not installed -> fall back to HTTP-only
	}
	return &playwrightRenderer{pw: pw}
}

func (r *playwrightRenderer) Render(ctx context.Context, url string) (string, int, error) {
	browser, err := r.pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
		Args:     []string{"--no-sandbox", "--disable-setuid-sandbox", "--disable-dev-shm-usage"},
	})
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = browser.Close() }()

	page, err := browser.NewPage(playwright.BrowserNewPageOptions{UserAgent: playwright.String(realisticUA)})
	if err != nil {
		return "", 0, err
	}
	resp, err := page.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(float64((25 * time.Second).Milliseconds())),
	})
	if err != nil {
		return "", 0, err
	}
	status := 0
	if resp != nil {
		status = resp.Status()
	}
	content, err := page.Content()
	if err != nil {
		return "", status, err
	}
	_ = ctx // Playwright manages its own timeouts
	return content, status, nil
}
