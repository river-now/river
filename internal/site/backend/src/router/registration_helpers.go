package router

import (
	"github.com/river-now/river"
)

type LoaderCtx struct{ *river.LoaderReqData }
type ActionCtx[I any] struct{ *river.ActionReqData[I] }

func decorateLoaderCtx(rd *river.LoaderReqData) *LoaderCtx {
	return &LoaderCtx{LoaderReqData: rd}
}
func decorateActionCtx[I any](rd *river.ActionReqData[I]) *ActionCtx[I] {
	return &ActionCtx[I]{ActionReqData: rd}
}

func NewLoader[O any](
	pattern string, loader river.LoaderFunc[LoaderCtx, O],
) *river.Loader[O] {
	return river.NewLoader(App, pattern, loader, decorateLoaderCtx)
}
func NewAction[I any, O any](
	method string, pattern string, action river.ActionFunc[ActionCtx[I], I, O],
) *river.Action[I, O] {
	return river.NewAction(App, method, pattern, action, decorateActionCtx)
}
