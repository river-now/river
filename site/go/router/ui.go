package router

import (
	"site/go/app"

	"github.com/sjc5/river/kit/mux"
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

var _ = newLoader("", func(c *mux.NestedReqData) (*RootData, error) {
	return &RootData{
		SiteTitle:     app.SiteTitle,
		LatestVersion: "v0.17.0-pre.12", // __TODO set this dynamically
	}, nil
})

var _ = newLoader("/", func(c *mux.NestedReqData) (string, error) {
	return app.SiteDescription, nil
})
