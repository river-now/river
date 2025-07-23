package router

import "github.com/river-now/river/kit/mux"

var LoadersRouter = mux.NewNestedRouter(&mux.NestedOptions{
	ExplicitIndexSegment: "_index", // Optional, but recommended
})

type LoaderCtx struct {
	*mux.NestedReqData
	// Anything else you want available on the LoaderCtx
}

func NewLoader[O any](pattern string, f func(c *LoaderCtx) (O, error)) *mux.TaskHandler[mux.None, O] {
	wrappedF := func(c *mux.NestedReqData) (O, error) {
		return f(&LoaderCtx{
			NestedReqData: c,
			// Anything else you want available on the LoaderCtx
		})
	}
	loaderTask := mux.TaskHandlerFromFunc(wrappedF)
	mux.RegisterNestedTaskHandler(LoadersRouter, pattern, loaderTask)
	return loaderTask
}

type RootData struct {
	Message string
}

// Example loader
var _ = NewLoader("/_index", func(c *LoaderCtx) (*RootData, error) {
	return &RootData{Message: "Hello from a River loader!"}, nil
})
