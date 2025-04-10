package router

import (
	"errors"
	"net/http"

	"github.com/sjc5/river/kit/mux"
	"github.com/sjc5/river/kit/validate"
)

var ActionsRouter = mux.NewRouter(&mux.Options{
	TasksRegistry: sharedTasksRegistry,
	MountRoot:     "/river-api/",
	MarshalInput: func(r *http.Request, iPtr any) error {
		if r.Method == http.MethodGet {
			return validate.URLSearchParamsInto(r, iPtr)
		}
		if r.Method == http.MethodPost {
			return validate.JSONBodyInto(r, iPtr)
		}
		return errors.New("unsupported method")
	},
	AutoTaskHandlerETags: true,
})

func newAction[I any, O any](method, pattern string, f mux.TaskHandlerFunc[I, O]) *mux.TaskHandler[I, O] {
	actionTask := mux.TaskHandlerFromFunc(ActionsRouter.TasksRegistry(), f)
	mux.RegisterTaskHandler(ActionsRouter, method, pattern, actionTask)
	return actionTask
}
