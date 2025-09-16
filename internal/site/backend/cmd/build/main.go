package main

import (
	"site/backend"
	"site/backend/src/router"

	"github.com/river-now/river"
)

func main() {
	backend.Wave.Builder(func(isDev bool) error {
		return backend.River.Build(&river.BuildOptions{
			IsDev:         isDev,
			LoadersRouter: router.LoadersRouter,
			ActionsRouter: router.ActionsRouter,
		})
	})
}
