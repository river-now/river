package router

import (
	"fmt"
	"strings"

	"site/app"
	"site/backend/markdown"

	"github.com/river-now/river"
	"github.com/river-now/river/kit/mux"
	"github.com/river-now/river/kit/xyz/fsmarkdown"
	"github.com/river-now/river/wave"
)

var LoadersRouter = mux.NewNestedRouter(&mux.NestedOptions{
	ExplicitIndexSegment: "_index",
})

func newLoader[O any](pattern string, f mux.TaskHandlerFunc[mux.None, O]) *mux.TaskHandler[mux.None, O] {
	loaderTask := mux.TaskHandlerFromFunc(f)
	mux.RegisterNestedTaskHandler(LoadersRouter, pattern, loaderTask)
	return loaderTask
}

type RootData struct {
	LatestVersion string
}

var currentNPMVersion = "v" + river.Internal__GetCurrentNPMVersion()

var jsonCacheControlVal = strings.Join([]string{
	"public",
	"max-age=60",                     // 1 minute in browser cache
	"s-maxage=86400",                 // 1 day in CDN cache
	"stale-while-revalidate=2592000", // 30 days stale in CDN while revalidating
	// skip "must-revalidate", as browsers seem to interpret it as though max-age=0
}, ", ")

var htmlCacheControlVal = strings.Join([]string{
	"public",
	"max-age=0",                      // no browser cache
	"s-maxage=86400",                 // 1 day in CDN cache
	"stale-while-revalidate=2592000", // 30 days stale in CDN while revalidating
	"must-revalidate",                // revalidate after 1 day in CDN
}, ", ")

var _ = newLoader("/", func(c *mux.NestedReqData) (*RootData, error) {
	r, rp := c.Request(), c.ResponseProxy()

	if !wave.GetIsDev() {
		// Because this app has no user-specific data, we can cache the responses
		// pretty aggressively.
		// Vercel purges the CDN on new deployments, so we don't need to worry about
		// build ID mismatches.
		if river.IsJSONRequest(r) {
			rp.SetHeader("Cache-Control", jsonCacheControlVal)
		} else {
			// Vary the HTML response by cookie to account for theme
			rp.SetHeader("Vary", "Cookie")
			rp.SetHeader("Cache-Control", htmlCacheControlVal)
		}
	}

	return &RootData{LatestVersion: currentNPMVersion}, nil
})

var _ = newLoader("/_index", func(c *mux.NestedReqData) (string, error) {
	return app.SiteDescription, nil
})

var _ = newLoader("/*", func(c *mux.NestedReqData) (*fsmarkdown.DetailedPage, error) {
	r, rp := c.Request(), c.ResponseProxy()

	p, err := markdown.Markdown.GetPageDetails(r)
	if err != nil {
		return nil, fmt.Errorf("failed to get page details: %w", err)
	}

	data := p
	e := river.NewHeadEls(2)

	if p.Title != "" {
		e.Title(fmt.Sprintf("%s | %s", app.SiteTitle, p.Title))
		e.Meta(e.Property("og:title"), e.Content(p.Title))
	}

	if p.Description != "" {
		e.Description(p.Description)
		e.Meta(e.Property("og:description"), e.Content(p.Description))
	}

	rp.AddHeadElements(e.Collect()...)

	return data, nil
})
