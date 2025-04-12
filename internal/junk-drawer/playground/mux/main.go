package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/river-now/river/kit/mux"
	"github.com/river-now/river/kit/tasks"
	"github.com/river-now/river/kit/validate"
)

var tasksRegistry = tasks.NewRegistry()

var r = mux.NewRouter(&mux.Options{
	TasksRegistry: tasksRegistry,
	MarshalInput:  validate.URLSearchParamsInto,
})

func main() {
	registerRoutes()

	server := &http.Server{Addr: ":9090", Handler: r}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			panic(err)
		}
	}()

	fmt.Println("Server running on port 9090")

	// hit a certain path on the running server

	resp, err := http.Get("http://localhost:9090/")
	if err != nil {
		panic(err)
	}

	fmt.Println("Response status:", resp.Status)

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	fmt.Println("Response body:", string(bodyText))
}

type None struct{}

func Get[I any, O any](pattern string, taskHandler *mux.TaskHandler[I, O]) {
	mux.RegisterTaskHandler(r, "GET", pattern, taskHandler)
}
func Post[I any, O any](pattern string, taskHandler *mux.TaskHandler[I, O]) {
	mux.RegisterTaskHandler(r, "POST", pattern, taskHandler)
}

/////////////////////////////////////////////////////////////////////////////////////////////

type Test struct {
	Input string `json:"input"`
}

var AuthTask = mux.TaskMiddlewareFromFunc(tasksRegistry, func(rd *mux.ReqData[mux.None]) (string, error) {
	fmt.Println("running auth ...", rd)
	res := rd.ResponseProxy()
	res.SetStatus(400)
	return "auth-token-43892", nil
})

// var _ = router.SetGlobalTaskMiddleware(r, AuthTask)

// var EmptyStrTaskHandler = mux.TaskHandlerFromFunc(tasksRegistry,
// 	func(rd *mux.ReqData[Test]) (string, error) {
// 		fmt.Println("running empty str ...", rd.Request().URL.Path)
// 		return "empty str", nil
// 	},
// )

func registerRoutes() {
	// Get("/", EmptyStrTaskHandler)
	x := mux.RegisterHandlerFunc(r, "GET", "/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("running slash ...", r.URL.Path)
		w.Write([]byte("slash"))
	})

	mux.SetRouteLevelTaskMiddleware(x, AuthTask)
}

// var _ = Get("/", func(rd *router.ReqData[Test]) (string, error) {
// 	fmt.Println("running slash ...", rd.Request().URL.Path)
// 	return "slash", nil
// })

// var sallyPattern = Get("/sally", func(rd *router.ReqData[string]) (string, error) {
// 	fmt.Println("running sally ...", rd)
// 	someInput := rd.Input()
// 	fmt.Println("running sally 2 ...", someInput)
// 	return "sally", nil
// })

// var catchAllRoute = Get("/*", func(rd *router.ReqData[Test]) (map[string]string, error) {
// 	input := rd.Input()
// 	tc := rd.TasksCtx()
// 	token, _ := AuthTask.Prep(tc, rd.Request()).Get()
// 	fmt.Println("Auth token from catch route:", token)

// 	fmt.Println("running hello ...", rd.SplatValues(), sallyPattern.Phantom)
// 	return map[string]string{
// 		"hello": "world",
// 		"foo":   input.Input,
// 	}, nil
// })

// var _ = mux.SetRouteLevelTaskMiddleware(catchAllRoute, AuthTask)
