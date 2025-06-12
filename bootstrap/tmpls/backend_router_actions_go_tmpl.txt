package router

import (
	"errors"
	"net/http"

	"github.com/river-now/river/kit/mux"
	"github.com/river-now/river/kit/validate"
)

var ActionsRouter = mux.NewRouter(&mux.Options{
	MountRoot:     "/api/",
	MarshalInput: func(r *http.Request, iPtr any) error {
		if r.Method == http.MethodGet {
			return validate.URLSearchParamsInto(r, iPtr)
		}
		if r.Method == http.MethodPost {
			return validate.JSONBodyInto(r, iPtr)
		}
		return errors.New("unsupported method")
	},
})

type ActionCtx[I any] struct {
	*mux.ReqData[I]
	// Anything else you want available on the ActionCtx
}

func NewAction[I any, O any](method, pattern string, f func(c *ActionCtx[I]) (O, error)) *mux.TaskHandler[I, O] {
	wrappedF := func(c *mux.ReqData[I]) (O, error) {
		return f(&ActionCtx[I]{
			ReqData: c,
			// Anything else you want available on the ActionCtx
		})
	}
	actionTask := mux.TaskHandlerFromFunc(wrappedF)
	mux.RegisterTaskHandler(ActionsRouter, method, pattern, actionTask)
	return actionTask
}
