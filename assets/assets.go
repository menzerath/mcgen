// Package assets bundles all required assets (images, fonts) into a single package.
package assets

import "embed"

// Backgrounds contains all background images used for the achievements.
//
//go:embed backgrounds/*.png
var Backgrounds embed.FS

// FontFile contains the font used for the achievement title and description.
//
//go:embed font.ttf
var FontFile []byte
