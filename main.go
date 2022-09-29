package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/f5yacobucci/clusterdebug/pkg/config"
	"github.com/f5yacobucci/clusterdebug/pkg/consensus"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

type ()

const (
	unknown = "debug-unknown"
)

func main() {
	e := echo.New()
	e.Validator = consensus.NewValidator()

	// Config
	conf := config.NewConfig()

	e.Logger.Printj(log.JSON{
		"message": "configuration set",
		"config":  fmt.Sprintf("%+v", conf),
	})

	// Setup
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cc := config.NewClusterContext(c, conf)
			return next(cc)
		}
	})
	consensus.RegisterEndpoints(e)

	// Run
	go func() {
		listener := fmt.Sprintf(":%d", conf.Port)
		if err := e.Start(listener); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatalj(log.JSON{
				"err":     err,
				"message": "shutting down the server",
			})
		}
	}()

	// "Registration" loop
	go consensus.Run(conf, e)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}
