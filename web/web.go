// Package web provides a web API for the generator.
package web

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/menzerath/mcgen/assets"
	"github.com/menzerath/mcgen/generator"
	"github.com/menzerath/mcgen/metrics"
	"github.com/prometheus/client_golang/prometheus"
	slogfiber "github.com/samber/slog-fiber"
)

// WebAPI provides a web API for the generator using the fiber framework.
type WebAPI struct {
	Generator *generator.Generator
}

// New returns a new WebAPI.
func New(generator *generator.Generator) WebAPI {
	return WebAPI{
		Generator: generator,
	}
}

// StartWebAPI starts the WebAPI, registers all routes and blocks until the server is shut down.
func (web WebAPI) StartWebAPI() {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: os.Getenv("MODE") == "production",
		ServerHeader:          "mcgen",
		ProxyHeader:           fiber.HeaderXForwardedFor,
	})

	// collect (but don't expose!) prometheus metrics
	fiberPrometheus := fiberprometheus.NewWithRegistry(prometheus.DefaultRegisterer, "", "http", "", nil)
	app.Use(fiberPrometheus.Middleware)

	// enable logging
	app.Use(slogfiber.NewWithConfig(slog.Default(), slogfiber.Config{
		DefaultLevel:     slog.LevelDebug,
		ClientErrorLevel: slog.LevelWarn,
		ServerErrorLevel: slog.LevelError,
	}))

	// register all routes
	app.Use("/", filesystem.New(filesystem.Config{
		Root:       http.FS(static),
		PathPrefix: "static",
	}))
	app.Get("/a.php", web.legacyAPIQuery)
	app.Get("/a/:background/:title/:text/*", web.legacyAPIPath)
	app.Get("/api/v1/achievement", web.achievementGet)
	app.Post("/api/v1/achievement", web.achievementPost)

	// register 404 handler
	app.Use(func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).SendString("Whatever you are looking for, it's not here ¯\\_(ツ)_/¯")
	})

	// enable a graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		_ = <-c

		slog.Info("stopping web api")
		if err := app.Shutdown(); err != nil {
			slog.Error("stopping web api failed", "error", err)
		}
	}()

	// determine the port to listen on
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// listen on configured port
	slog.Info("web api starting", "port", port)
	if err := app.Listen(fmt.Sprintf(":%s", port)); err != nil {
		slog.Error("web api listening", "error", err)
	}
	slog.Warn("web api stopped")
}

func (web WebAPI) legacyAPIQuery(c *fiber.Ctx) error {
	// map the legacy icon ID to the new background name
	background, _ := assets.LegacyIconMappings[c.Query("i")]

	// decide on the output type
	output := AchievementOutputTypeDefault
	if c.Query("d") == "1" {
		output = AchievementOutputTypeDownload
	}

	return web.generateAndReturnAchievement(
		c,
		AchievementRequest{
			Background: background,
			Title:      c.Query("h"),
			Text:       c.Query("t"),
			Output:     output,
		},
	)
}

func (web WebAPI) legacyAPIPath(c *fiber.Ctx) error {
	// map the legacy icon ID to the new background name
	background, _ := assets.LegacyIconMappings[c.Params("background")]

	// decode the title and text
	title, err := url.QueryUnescape(c.Params("title"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   err.Error(),
			Message: "invalid title",
		})
	}
	text, err := url.QueryUnescape(c.Params("text"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   err.Error(),
			Message: "invalid text",
		})
	}

	return web.generateAndReturnAchievement(
		c,
		AchievementRequest{
			Background: background,
			Title:      title,
			Text:       text,
			Output:     AchievementOutputTypeDefault,
		},
	)
}

func (web WebAPI) achievementGet(c *fiber.Ctx) error {
	return web.generateAndReturnAchievement(
		c,
		AchievementRequest{
			Background: c.Query("background"),
			Title:      c.Query("title"),
			Text:       c.Query("text"),
			Output:     AchievementOutputType(c.Query("output")),
		},
	)
}

func (web WebAPI) achievementPost(c *fiber.Ctx) error {
	var request AchievementRequest
	if err := c.BodyParser(request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   err.Error(),
			Message: "invalid request body",
		})
	}

	return web.generateAndReturnAchievement(c, request)
}

func (web WebAPI) generateAndReturnAchievement(c *fiber.Ctx, request AchievementRequest) error {
	timeStart := time.Now()
	achievement, err := web.Generator.Generate(request.Background, request.Title, request.Text)
	if err != nil {
		if err == generator.ErrUnknownBackground {
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Error:   err.Error(),
				Message: "unknown background",
			})
		}

		slog.Error("generating image", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error:   err.Error(),
			Message: "could not generate achievement",
		})
	}
	metrics.AchievementGenerationRuntime.Observe(time.Now().Sub(timeStart).Seconds())
	slog.Info(
		"generated image",
		"background", request.Background,
		"title", request.Title,
		"text", request.Text,
		"runtime", time.Now().Sub(timeStart).Seconds(),
	)

	// return image as download
	if request.Output == AchievementOutputTypeDownload {
		c.Set("Content-Description", "File Transfer")
		c.Set("Content-Type", "application/octet-image")
		c.Set("Content-Disposition", "attachment; filename=achievement.png")
		return c.Status(fiber.StatusOK).Send(achievement)
	}

	// return image in response
	c.Type("png")
	return c.Status(fiber.StatusOK).Send(achievement)
}
