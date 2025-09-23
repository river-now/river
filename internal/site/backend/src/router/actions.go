package router

import "github.com/river-now/river"

var ActionsRouter = river.NewActionsRouter()

type ActionCtx[I any] struct {
	*river.ActionReqData[I]
}

func actionCtxFactory[I any](rd *river.ActionReqData[I]) *ActionCtx[I] {
	return &ActionCtx[I]{
		ActionReqData: rd,
	}
}

func NewAction[I any, O any](
	method string,
	pattern string,
	actionFunc river.ActionFunc[ActionCtx[I], I, O],
) *river.Action[I, O] {
	return river.NewAction(ActionsRouter, method, pattern, actionFunc, actionCtxFactory)
}
