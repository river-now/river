package server

import (
	"app/app"
	"app/backend/router"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/river-now/river/kit/grace"
	"github.com/river-now/river/wave"
)

func Serve() {
	app.River.Init(wave.GetIsDev())

	addr := fmt.Sprintf(":%d", wave.MustGetPort())

	server := &http.Server{
		Addr:                         addr,
		Handler:                      http.TimeoutHandler(router.Core(), 60*time.Second, "Request timed out"),
		ReadTimeout:                  15 * time.Second,
		WriteTimeout:                 30 * time.Second,
		IdleTimeout:                  60 * time.Second,
		ReadHeaderTimeout:            10 * time.Second,
		MaxHeaderBytes:               1 << 20, // 1 MB
		DisableGeneralOptionsHandler: true,
		ErrorLog:                     log.New(os.Stderr, "HTTP: ", log.Ldate|log.Ltime|log.Lshortfile),
	}

	url := "http://localhost" + addr

	grace.Orchestrate(grace.OrchestrateOptions{
		StartupCallback: func() error {
			app.Log.Info("Starting server", "url", url)

			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Server listen and serve error: %v\n", err)
			}

			return nil
		},

		ShutdownCallback: func(shutdownCtx context.Context) error {
			app.Log.Info("Shutting down server", "url", url)

			if err := server.Shutdown(shutdownCtx); err != nil {
				log.Fatalf("Server shutdown error: %v\n", err)
			}

			return nil
		},
	})
}
