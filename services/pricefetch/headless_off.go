package pricefetch

// NewHeadless returns nil: no headless browser is linked by default, so the
// standard build and Docker image stay lean (HTTP fetching only).
//
// Headless rendering of JS-heavy price pages is an extension point: implement a
// Renderer with a browser automation library (e.g. playwright-go, Chromium with
// --no-sandbox for containers) and pass it to NewHTTPSource/DefaultSources in
// the router. The source layer already falls back to the headless Renderer when
// an HTTP fetch returns an empty/JS-only page.
func NewHeadless() Renderer { return nil }
