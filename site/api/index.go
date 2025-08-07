package api

import (
	"app/app"
	"app/backend/router"
	"net/http"

	"github.com/river-now/river/wave"
)

var appRouter = router.Core()

func init() {
	app.River.Init(wave.GetIsDev())
}

func Handler(w http.ResponseWriter, r *http.Request) {
	appRouter.ServeHTTP(w, r)
}
