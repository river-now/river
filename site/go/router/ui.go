package router

import (
	"site/go/app"

	"github.com/river-now/river"
	"github.com/river-now/river/kit/mux"
)

var UIRouter = mux.NewNestedRouter(&mux.NestedOptions{TasksRegistry: sharedTasksRegistry})

func newLoader[O any](pattern string, f mux.TaskHandlerFunc[mux.None, O]) *mux.TaskHandler[mux.None, O] {
	loaderTask := mux.TaskHandlerFromFunc(UIRouter.TasksRegistry(), f)
	mux.RegisterNestedTaskHandler(UIRouter, pattern, loaderTask)
	return loaderTask
}

type RootData struct {
	SiteTitle     string
	LatestVersion string
}

var currentNPMVersion = "v" + river.Internal__GetCurrentNPMVersion()

var _ = newLoader("", func(c *mux.NestedReqData) (*RootData, error) {
	return &RootData{
		SiteTitle:     app.SiteTitle,
		LatestVersion: currentNPMVersion,
	}, nil
})

var _ = newLoader("/", func(c *mux.NestedReqData) (string, error) {
	return app.SiteDescription, nil
})
