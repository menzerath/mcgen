package web

import (
	"embed"
	"fmt"
	"io/fs"
)

// frontend contains the files for the simple UI served by the web server on the base URL.
// Use the [FrontendFS] function to get a filesystem suitable for use with the fiber static middleware.
//
//go:embed static/*
var frontend embed.FS

// FrontendFS returns the embedded filesystem for the frontend static files, stripping the "static" prefix.
// It panics if the embedded filesystem cannot be subsetted.
func FrontendFS() fs.FS {
	frontendFS, err := fs.Sub(frontend, "static")
	if err != nil {
		panic(fmt.Errorf("subsetting embedded frontend filesystem: %w", err))
	}
	return frontendFS
}
