// Package main provides the entrypoint for mcgen.
package main

import (
	"log/slog"
	"os"

	"github.com/menzerath/mcgen/generator"
	"github.com/menzerath/mcgen/metrics"
	"github.com/menzerath/mcgen/web"
)

func main() {
	initLogging()
	go metrics.ExposeMetrics()

	gen, err := generator.New()
	if err != nil {
		slog.Error("initializing generator", "error", err)
		os.Exit(1)
	}

	webAPI := web.New(gen)
	webAPI.StartWebAPI()
}

func initLogging() {
	var handler slog.Handler
	if os.Getenv("MODE") == "production" {
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	} else {
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelDebug,
		})
	}
	slog.SetDefault(slog.New(handler))
}
