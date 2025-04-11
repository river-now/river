package router

import (
	"fmt"
	"site/go/app"

	"github.com/sjc5/river/kit/mux"
)

var UIRouter = mux.NewNestedRouter(&mux.NestedOptions{TasksRegistry: sharedTasksRegistry})

func newLoader[O any](pattern string, f mux.TaskHandlerFunc[mux.None, O]) *mux.TaskHandler[mux.None, O] {
	loaderTask := mux.TaskHandlerFromFunc(UIRouter.TasksRegistry(), f)
	mux.RegisterNestedTaskHandler(UIRouter, pattern, loaderTask)
	return loaderTask
}

var _ = newLoader("", func(c *mux.NestedReqData) (string, error) {
	r := c.Request()
	match := r.Header.Get("If-None-Match")
	fmt.Println(r.URL.Path, "If-None-Match:", match)
	return app.SiteTitle, nil
})

var _ = newLoader("/", func(c *mux.NestedReqData) (string, error) {
	return app.SiteDescription, nil
})
