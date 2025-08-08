package api

import (
	"net/http"

	"site/app"
	"site/backend/router"

	"github.com/river-now/river/wave"
)

var siteRouter = router.Core()

func init() {
	app.River.Init(wave.GetIsDev())
}

func Handler(w http.ResponseWriter, r *http.Request) {
	siteRouter.ServeHTTP(w, r)
}
