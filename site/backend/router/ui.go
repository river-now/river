package router

import (
	"fmt"
	app "site"
	"site/backend/markdown"

	"github.com/river-now/river"
	"github.com/river-now/river/kit/mux"
	"github.com/river-now/river/kit/xyz/fsmarkdown"
	"github.com/river-now/river/wave"
)

var UIRouter = mux.NewNestedRouter(&mux.NestedOptions{TasksRegistry: sharedTasksRegistry})

func newLoader[O any](pattern string, f mux.TaskHandlerFunc[mux.None, O]) *mux.TaskHandler[mux.None, O] {
	loaderTask := mux.TaskHandlerFromFunc(UIRouter.TasksRegistry(), f)
	mux.RegisterNestedTaskHandler(UIRouter, pattern, loaderTask)
	return loaderTask
}

type RootData struct {
	LatestVersion string
}

var currentNPMVersion = "v" + river.Internal__GetCurrentNPMVersion()

var _ = newLoader("", func(c *mux.NestedReqData) (*RootData, error) {
	r := c.Request()

	// We don't want to cache HTML, so that user theme settings do not get
	// CDN-cached, and to ensure that fresh page loads always get the latest
	// version of the site. But we can cache JSON requests because no data
	// returned in the JSON for this site is user-specific. However, because
	// the build ID serves as a cache discriminator (e.g., river_json=1234),
	// we don't want to cache requests with stale build IDs from this build.
	if app.River.IsCurrentBuildJSONRequest(r) && !wave.GetIsDev() {
		c.ResponseProxy().SetHeader("Cache-Control", "public, max-age=5, must-revalidate")
	}

	return &RootData{
		LatestVersion: currentNPMVersion,
	}, nil
})

var _ = newLoader("/", func(c *mux.NestedReqData) (string, error) {
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
