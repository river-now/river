package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/river-now/river/kit/tasks"
)

var tasksRegistry = tasks.NewRegistry("jd")

var (
	Auth    = tasks.Register(tasksRegistry, auth)
	User    = tasks.Register(tasksRegistry, user)
	User2   = tasks.Register(tasksRegistry, user2)
	Profile = tasks.Register(tasksRegistry, profile)
)

func main() {
	go func() {
		// every 1 second, print a new line
		for {
			fmt.Println()
			time.Sleep(1 * time.Second)
		}
	}()

	req, _ := http.NewRequest("GET", "http://localhost:8080", nil)
	c := tasksRegistry.NewCtxFromRequest(req)

	data, err := Profile.Prep(c, "32isdoghj").Get()

	fmt.Println("from main -- profile data:", data)
	fmt.Println("from main -- profile err:", err)
}

func auth(c *tasks.ArgNoInput) (int, error) {
	fmt.Println("running auth   ...", c.Request().URL, time.Now().UnixMilli())
	// return 0, errors.New("auth error")

	time.Sleep(2 * time.Second)

	fmt.Println("auth done", time.Now().UnixMilli())
	return 123, nil
}

func user(c *tasks.Arg[string]) (string, error) {
	user_id := c.Input
	fmt.Println("running user   ...", user_id, time.Now().UnixMilli())
	// time.Sleep(500 * time.Millisecond)
	// c.Cancel()
	token, _ := Auth.PrepNoInput(c.TasksCtx).Get()
	fmt.Println("user retrieved token", token)

	time.Sleep(2 * time.Second)

	fmt.Println("user done", time.Now().UnixMilli())
	return fmt.Sprintf("user-%d", token), nil
}

func user2(c *tasks.Arg[string]) (string, error) {
	fmt.Println("running user2  ...", c.Request().URL, time.Now().UnixMilli())
	token, _ := Auth.PrepNoInput(c.TasksCtx).Get()
	fmt.Println("user2 retrieved token", token)

	time.Sleep(2 * time.Second)

	fmt.Println("user2 done", time.Now().UnixMilli())
	return fmt.Sprintf("user2-%d", token), nil
}

func profile(c *tasks.Arg[string]) (string, error) {
	user_id := c.Input
	fmt.Println("running profile...", c.Request().URL, time.Now().UnixMilli())
	user := User.Prep(c.TasksCtx, user_id)
	user2 := User2.Prep(c.TasksCtx, user_id)

	fmt.Println("profile running user, user2 in parallel")
	if ok := c.ParallelPreload(user, user2); !ok {
		return "", errors.New("user error")
	}
	fmt.Println("profile running user, user2 in parallel done")
	userData, _ := user.Get()
	user2Data, _ := user2.Get()
	fmt.Println("profile user, user2 data", userData, user2Data)

	time.Sleep(2 * time.Second)
	fmt.Println("profile done", time.Now().UnixMilli(), userData, user2Data)
	return "profile", nil
}
