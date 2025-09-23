package main

import (
	"site/backend"
	"site/backend/src/router"

	"github.com/river-now/river"
)

func main() {
	backend.App.BuildRiver(river.BuildRiverOptions{
		LoadersRouter: router.LoadersRouter,
		ActionsRouter: router.ActionsRouter,
	})
}
