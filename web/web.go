// Package web provides a web API for the generator.
package web

import (
	"log/slog"
	"os"
	"os/signal"

	"github.com/gofiber/fiber/v2"
	"github.com/menzerath/mcgen/assets"
	"github.com/menzerath/mcgen/generator"
)

// WebAPI provides a web API for the generator using the fiber framework.
type WebAPI struct {
	Generator generator.Generator
}

// New returns a new WebAPI.
func New(generator generator.Generator) WebAPI {
	return WebAPI{
		Generator: generator,
	}
}

// StartWebAPI starts the WebAPI, registers all routes and blocks until the server is shut down.
func (web WebAPI) StartWebAPI() {
	app := fiber.New()

	// register all routes
	app.Get("/a.php", web.legacyAPI)
	app.Get("/api/v1/achievement", web.achievementGet)
	app.Post("/api/v1/achievement", web.achievementPost)

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

	// listen on port 8080
	if err := app.Listen(":8080"); err != nil {
		slog.Error("web api listening", "error", err)
	}
	slog.Warn("web api stopped")
}

func (web WebAPI) legacyAPI(c *fiber.Ctx) error {
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
		return c.Status(400).JSON(ErrorResponse{
			Error:   err.Error(),
			Message: "invalid request body",
		})
	}

	return web.generateAndReturnAchievement(c, request)
}

func (web WebAPI) generateAndReturnAchievement(c *fiber.Ctx, request AchievementRequest) error {
	achievement, err := web.Generator.Generate(request.Background, request.Title, request.Text)
	if err != nil {
		if err == generator.ErrUnknownBackground {
			return c.Status(400).JSON(ErrorResponse{
				Error:   err.Error(),
				Message: "unknown background",
			})
		}
		return c.Status(500).JSON(ErrorResponse{
			Error:   err.Error(),
			Message: "could not generate achievement",
		})
	}

	// return image as download
	if request.Output == AchievementOutputTypeDownload {
		c.Set("Content-Description", "File Transfer")
		c.Set("Content-Type", "application/octet-image")
		c.Set("Content-Disposition", "attachment; filename=achievement.png")
		return c.Status(200).Send(achievement)
	}

	// return image in response
	c.Type("png")
	return c.Status(200).Send(achievement)
}
