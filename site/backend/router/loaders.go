package router

import (
	"app"
	"app/backend/markdown"
	"fmt"

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

var _ = newLoader("/", func(c *mux.NestedReqData) (*RootData, error) {
	req, res := c.Request(), c.ResponseProxy()

	if !wave.GetIsDev() {
		if river.IsJSONRequest(req) {
			// Because this app has no user-specific data, we can cache the JSON response
			// pretty aggressively.
			// res.SetHeader("Cache-Control", "public, max-age=10, s-maxage=20, must-revalidate")
		} else {
			// Don't cache HTML, but stop short of "no-store" so it's still eligible for ETag revalidation
			res.SetHeader("Cache-Control", "no-cache")
		}
	}

	return &RootData{LatestVersion: currentNPMVersion}, nil
})

var _ = newLoader("/_index", func(c *mux.NestedReqData) (string, error) {
	return app.SiteDescription, nil
})

var _ = newLoader("/*", func(c *mux.NestedReqData) (*fsmarkdown.DetailedPage, error) {
	r := c.Request()

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

	c.ResponseProxy().AddHeadElements(e.Collect()...)

	return data, nil
})
