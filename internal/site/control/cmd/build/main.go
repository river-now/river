package main

import (
	"site/backend/router"
	"site/control"

	"github.com/river-now/river"
)

func main() {
	control.Wave.Builder(func(isDev bool) error {
		return control.River.Build(&river.BuildOptions{
			IsDev:         isDev,
			LoadersRouter: router.LoadersRouter,
			ActionsRouter: router.ActionsRouter,
		})
	})
}
