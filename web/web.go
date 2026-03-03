// Package web provides a web API for the generator.
package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/menzerath/mcgen/assets"
	"github.com/menzerath/mcgen/generator"
	"github.com/menzerath/mcgen/metrics"
)

// WebAPI provides a web API for the generator.
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
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Server", "mcgen")
			next.ServeHTTP(w, r)
		})
	})
	r.Use(prometheusMiddleware)
	r.Use(slogLoggingMiddleware)

	// register API routes
	r.Get("/a.php", web.legacyAPIQuery)
	r.Get("/a/{background}/{title}/{text}", web.legacyAPIPath)
	r.Get("/api/v1/achievement", web.achievementGet)
	r.Post("/api/v1/achievement", web.achievementPost)

	// serve embedded static files for requests that don't match any API route
	subFS, err := fs.Sub(static, "static")
	if err != nil {
		slog.Error("creating static sub-filesystem", "error", err)
		os.Exit(1)
	}
	fileServer := http.FileServer(http.FS(subFS))

	// serve static files; fall back to a custom 404 for anything else
	r.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "."
		}
		if _, err := subFS.Open(path); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}
		http.Error(w, `Whatever you are looking for, it's not here ¯\_(ツ)_/¯`, http.StatusNotFound)
	}))

	// determine the port to listen on
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: r,
	}

	// enable a graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		_ = <-c

		slog.Info("stopping web api")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			slog.Error("stopping web api failed", "error", err)
		}
	}()

	// listen on configured port
	slog.Info("web api starting", "port", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("web api listening", "error", err)
	}
	slog.Warn("web api stopped")
}

// writeJSON writes v as JSON with the given HTTP status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (web WebAPI) legacyAPIQuery(w http.ResponseWriter, r *http.Request) {
	// map the legacy icon ID to the new background name
	background, _ := assets.LegacyIconMappings[r.URL.Query().Get("i")]

	// decide on the output type
	output := AchievementOutputTypeDefault
	if r.URL.Query().Get("d") == "1" {
		output = AchievementOutputTypeDownload
	}

	web.generateAndReturnAchievement(w, r, AchievementRequest{
		Background: background,
		Title:      r.URL.Query().Get("h"),
		Text:       r.URL.Query().Get("t"),
		Output:     output,
	})
}

func (web WebAPI) legacyAPIPath(w http.ResponseWriter, r *http.Request) {
	// map the legacy icon ID to the new background name
	background, _ := assets.LegacyIconMappings[chi.URLParam(r, "background")]

	// decode the title and text
	title, err := url.QueryUnescape(chi.URLParam(r, "title"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error:   err.Error(),
			Message: "invalid title",
		})
		return
	}
	text, err := url.QueryUnescape(chi.URLParam(r, "text"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error:   err.Error(),
			Message: "invalid text",
		})
		return
	}

	web.generateAndReturnAchievement(w, r, AchievementRequest{
		Background: background,
		Title:      title,
		Text:       text,
		Output:     AchievementOutputTypeDefault,
	})
}

func (web WebAPI) achievementGet(w http.ResponseWriter, r *http.Request) {
	web.generateAndReturnAchievement(w, r, AchievementRequest{
		Background: r.URL.Query().Get("background"),
		Title:      r.URL.Query().Get("title"),
		Text:       r.URL.Query().Get("text"),
		Output:     AchievementOutputType(r.URL.Query().Get("output")),
	})
}

func (web WebAPI) achievementPost(w http.ResponseWriter, r *http.Request) {
	var request AchievementRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error:   err.Error(),
			Message: "invalid request body",
		})
		return
	}

	web.generateAndReturnAchievement(w, r, request)
}

func (web WebAPI) generateAndReturnAchievement(w http.ResponseWriter, r *http.Request, request AchievementRequest) {
	timeStart := time.Now()
	achievement, err := web.Generator.Generate(request.Background, request.Title, request.Text)
	if err != nil {
		if err == generator.ErrUnknownBackground {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{
				Error:   err.Error(),
				Message: "unknown background",
			})
			return
		}

		slog.Error("generating image", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{
			Error:   err.Error(),
			Message: "could not generate achievement",
		})
		return
	}
	metrics.AchievementGenerationRuntime.Observe(time.Since(timeStart).Seconds())
	slog.Info(
		"generated image",
		"background", request.Background,
		"title", request.Title,
		"text", request.Text,
		"runtime", time.Since(timeStart).Seconds(),
	)

	// return image as download
	if request.Output == AchievementOutputTypeDownload {
		w.Header().Set("Content-Description", "File Transfer")
		w.Header().Set("Content-Type", "application/octet-image")
		w.Header().Set("Content-Disposition", "attachment; filename=achievement.png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(achievement)
		return
	}

	// return image in response
	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(achievement)
}
