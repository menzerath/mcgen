// Package generator handles the generation of achievement images.
package generator

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log/slog"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"github.com/menzerath/mcgen/assets"
	"golang.org/x/image/font"
)

// list of errors returned by the generator
var (
	ErrUnknownBackground = fmt.Errorf("unknown background")
)

// A Generator manages all resources required to generate achievement images and provides a method to generate them.
type Generator struct {
	Backgrounds map[string]image.Image
	FontFace    font.Face
}

// New returns a new generator.
// It loads all embedded assets for quick access when needed.
func New() (Generator, error) {
	generator := Generator{
		Backgrounds: make(map[string]image.Image),
	}

	// read all embedded background files and put them into our generator's map
	files, err := assets.Backgrounds.ReadDir("backgrounds")
	if err != nil {
		return Generator{}, fmt.Errorf("reading backgrounds: %w", err)
	}
	for _, file := range files {
		content, err := assets.Backgrounds.ReadFile(fmt.Sprintf("backgrounds/%s", file.Name()))
		if err != nil {
			return Generator{}, fmt.Errorf("reading background %s: %w", file.Name(), err)
		}

		background, _, err := image.Decode(bytes.NewReader(content))
		if err != nil {
			return Generator{}, fmt.Errorf("decoding background %s: %w", file.Name(), err)
		}

		generator.Backgrounds[file.Name()] = background
		slog.Debug("loaded background", "name", file.Name())
	}
	slog.Debug("loaded all backgrounds", "count", len(generator.Backgrounds))

	// parse the font and store it in our generator
	parsedFont, err := truetype.Parse(assets.FontFile)
	if err != nil {
		return Generator{}, fmt.Errorf("parsing font: %w", err)
	}
	generator.FontFace = truetype.NewFace(
		parsedFont,
		&truetype.Options{
			Size:    16,
			Hinting: font.HintingFull,
		},
	)
	slog.Debug("loaded font")

	return generator, nil
}

// Generate generates an achievement image with the given background and text.
// It will return an error if the background is unknown.
func (generator Generator) Generate(background string, textTop string, textBottom string) ([]byte, error) {
	slog.Info("generating image", "background", background, "textTop", textTop, "textBottom", textBottom)

	// load background template
	template, exists := generator.Backgrounds[fmt.Sprintf("%s.png", background)]
	if !exists {
		return nil, ErrUnknownBackground
	}

	dc := gg.NewContextForImage(template)
	dc.SetFontFace(generator.FontFace)

	// write text on background
	dc.SetColor(color.RGBA{R: 255, G: 255, B: 0, A: 255})
	dc.DrawString(textTop, 60, 28)

	dc.SetColor(color.RGBA{R: 255, G: 255, B: 255, A: 255})
	dc.DrawString(textBottom, 60, 50)

	// encode image as bytes
	buffer := new(bytes.Buffer)
	if err := png.Encode(buffer, dc.Image()); err != nil {
		return nil, fmt.Errorf("encoding image: %w", err)
	}
	return buffer.Bytes(), nil
}
