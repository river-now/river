package main

import (
	"app/app"
	"app/backend/router"

	"github.com/river-now/river"
	"github.com/river-now/river/kit/tsgen"
)

func main() {
	a := tsgen.Statements{}

	a.Serialize("export const ACTIONS_ROUTER_MOUNT_ROOT", router.ActionsRouter.MountRoot())

	app.Wave.Builder(func(isDev bool) error {
		return app.River.Build(&river.BuildOptions{
			IsDev:         isDev,
			LoadersRouter: router.LoadersRouter,
			ActionsRouter: router.ActionsRouter,
			AdHocTypes:    []*river.AdHocType{},
			ExtraTSCode:   a.BuildString(),
		})
	})
}
