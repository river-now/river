package mux

import (
	"github.com/river-now/river/kit/genericsutil"
	"github.com/river-now/river/kit/matcher"
	"github.com/river-now/river/kit/response"
	"github.com/river-now/river/kit/tasks"
)

type (
	None                      = genericsutil.None
	TaskHandler[I any, O any] = tasks.RegisteredTask[*ReqData[I], O]
	Params                    = matcher.Params
)

type ReqData[I any] struct {
	_params         Params
	_splat_vals     []string
	_tasks_ctx      *tasks.TasksCtx
	_input          I
	_response_proxy *response.Proxy
}
