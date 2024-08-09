package web

import "embed"

// static contains the files for the simple UI served by the web server on the base URL.
//
//go:embed static/*
var static embed.FS
