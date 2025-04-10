package main

import (
	"site/go/app"
	"site/go/router"

	"github.com/sjc5/river"
	"github.com/sjc5/river/kit/tsgen"
)

func main() {
	a := tsgen.Statements{}

	a.Serialize("export const ACTIONS_ROUTER_MOUNT_ROOT", router.ActionsRouter.MountRoot())

	app.Kiruna.BuildHelper(func(isDev bool) error {
		return app.River.Build(&river.BuildOptions{
			IsDev:         isDev,
			UIRouter:      router.UIRouter,
			ActionsRouter: router.ActionsRouter,
			AdHocTypes:    []*river.AdHocType{},
			ExtraTSCode:   a.BuildString(),
		})
	})
}
