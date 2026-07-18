package pico8

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

// availableSections returns how many 128x32 sprite sections are exported, based
// on which dual-purpose sections are reused for the map.
func availableSections(useSection3, useSection4 bool) int {
	switch {
	case !useSection3 && !useSection4:
		return 4 // All sections available
	case !useSection3 || !useSection4:
		return 3 // Either Section 3 or Section 4 available
	default:
		return 2 // Only the base section available
	}
}

// reconstructImage puts the 16x16 sprite data into an RGBA image
func reconstructImage(gfxData []string) *image.RGBA {
	const size = 16 * 8 // 128
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	for y, line := range gfxData {
		for x, hexChar := range line {
			colorIndex := parseHexChar(hexChar)
			if colorIndex >= 0 && colorIndex < len(pico8Palette) {
				img.Set(x, y, pico8Palette[colorIndex])
			} else {
				// If out of range or invalid hex, default to black
				img.Set(x, y, pico8Palette[0])
			}
		}
	}

	return img
}

// renderMap draws the map data (and dual-purpose sections) onto a new RGBA
func renderMap(
	mapData []string,
	dualPurposeSection1, dualPurposeSection2 []string,
	spriteSheet *image.RGBA,
	mapWidth, mapHeight int,
) *image.RGBA {
	const tileSize = 8
	mapImage := image.NewRGBA(image.Rect(0, 0, mapWidth*tileSize, mapHeight*tileSize))

	// Fill background with black
	for y := 0; y < mapHeight*tileSize; y++ {
		for x := 0; x < mapWidth*tileSize; x++ {
			mapImage.Set(x, y, pico8Palette[0])
		}
	}

	// Draw regular map data (each line is hex data for 128 tiles = 256 hex chars)
	for y := 0; y < len(mapData); y++ {
		line := mapData[y]
		// Each tile is 2 hex digits => 1 tile
		for x := 0; x < len(line)/2; x++ {
			spriteY := parseHexChar(rune(line[x*2]))   // First hex digit is spriteY
			spriteX := parseHexChar(rune(line[x*2+1])) // Second hex digit is spriteX

			if spriteX != 0 || spriteY != 0 {
				drawSprite(mapImage, spriteSheet, spriteX, spriteY, x, y)
			}
		}
	}

	// Now draw dual-purpose section 1 (section 3)
	if dualPurposeSection1 != nil {
		for y := 0; y < len(dualPurposeSection1); y++ {
			line := dualPurposeSection1[y]
			for x := 0; x < len(line)/2; x++ {
				spriteX := parseHexChar(rune(line[x*2]))
				spriteY := parseHexChar(rune(line[x*2+1]))

				yIsEven := (y % 2) == 0
				if yIsEven {
					// same logic as your original "if yIsOdd" (slight rename for clarity)
					if spriteX != 0 || spriteY != 0 {
						drawSprite(mapImage, spriteSheet, spriteX, spriteY, x, 32+(y/2))
					} else {
						drawBlackTile(mapImage, x, 32+(y/2))
					}
				} else {
					if spriteX != 0 || spriteY != 0 {
						drawSprite(mapImage, spriteSheet, spriteX, spriteY, 64+x, 32+((y-1)/2))
					}
				}
			}
		}
	}

	// Dual-purpose section 2 (section 4)
	if dualPurposeSection2 != nil {
		for y := 0; y < len(dualPurposeSection2); y++ {
			line := dualPurposeSection2[y]
			for x := 0; x < len(line)/2; x++ {
				spriteX := parseHexChar(rune(line[x*2]))
				spriteY := parseHexChar(rune(line[x*2+1]))

				yIsEven := (y % 2) == 0
				if yIsEven {
					if spriteX != 0 || spriteY != 0 {
						drawSprite(mapImage, spriteSheet, spriteX, spriteY, x, 48+(y/2))
					}
				} else {
					if spriteX != 0 || spriteY != 0 {
						drawSprite(mapImage, spriteSheet, spriteX, spriteY, 64+x, 48+((y-1)/2))
					}
				}
			}
		}
	}

	return mapImage
}

// drawSprite copies an 8x8 region from the sprite sheet
func drawSprite(dst *image.RGBA, src *image.RGBA, spriteX, spriteY, dstTileX, dstTileY int) {
	const tileSize = 8
	srcX := spriteX * tileSize
	srcY := spriteY * tileSize
	dstX := dstTileX * tileSize
	dstY := dstTileY * tileSize

	for yy := 0; yy < tileSize; yy++ {
		for xx := 0; xx < tileSize; xx++ {
			dst.Set(dstX+xx, dstY+yy, src.At(srcX+xx, srcY+yy))
		}
	}
}

// drawBlackTile just fills an 8x8 region with black
func drawBlackTile(dst *image.RGBA, tileX, tileY int) {
	const tileSize = 8
	dstX := tileX * tileSize
	dstY := tileY * tileSize

	for yy := 0; yy < tileSize; yy++ {
		for xx := 0; xx < tileSize; xx++ {
			dst.Set(dstX+xx, dstY+yy, pico8Palette[0])
		}
	}
}

// saveSprites writes the sprite sub-image sections (each 128x32) into outDir/sprites.
func saveSprites(spriteSheet *image.RGBA, outDir string, useSection3, useSection4 bool) error {
	const tileSize = 8

	numSections := availableSections(useSection3, useSection4)

	spritesDir := filepath.Join(outDir, "sprites")
	if err := os.MkdirAll(spritesDir, 0755); err != nil {
		return fmt.Errorf("failed to create sprites/ dir: %w", err)
	}

	// Save the sub-image sections (each 128x32): section_0.png .. section_{n-1}.png
	const subImageHeight = 4 * tileSize // 32 px
	const subImageWidth = 16 * tileSize // 128 px

	for i := 0; i < numSections; i++ {
		subImg := image.NewRGBA(image.Rect(0, 0, subImageWidth, subImageHeight))
		startY := i * subImageHeight

		// Copy from spriteSheet, checking bounds for robustness
		copyHeight := subImageHeight
		if startY+copyHeight > spriteSheet.Bounds().Dy() {
			copyHeight = spriteSheet.Bounds().Dy() - startY
		}
		copyWidth := subImageWidth
		if copyWidth > spriteSheet.Bounds().Dx() {
			copyWidth = spriteSheet.Bounds().Dx()
		}

		if copyHeight > 0 && copyWidth > 0 {
			for yy := 0; yy < copyHeight; yy++ {
				for xx := 0; xx < copyWidth; xx++ {
					subImg.Set(xx, yy, spriteSheet.At(xx, startY+yy))
				}
			}
		}

		subImagePath := filepath.Join(spritesDir, fmt.Sprintf("section_%d.png", i))
		if err := saveAsPng(subImg, subImagePath); err != nil {
			return fmt.Errorf("error saving %s: %w", subImagePath, err)
		}
	}

	return nil
}

// saveAsPng encodes the RGBA image to a PNG file
func saveAsPng(img *image.RGBA, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck

	return png.Encode(f, img)
}

// loadImage loads an image from a file path
func loadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "error closing file %s: %v\n", path, cerr)
		}
	}()

	img, err := png.Decode(f)
	if err != nil {
		return nil, err
	}

	return img, nil
}

// combineSectionsIntoSpriteSheet combines the section images in outDir/sprites
// into outDir/spritesheet.png.
func combineSectionsIntoSpriteSheet(outDir string, numSections int) error {
	const sectionWidth = 128
	const sectionHeight = 32

	// Create a new image to hold the combined sprite sheet (always full height)
	combined := image.NewRGBA(image.Rect(0, 0, sectionWidth, sectionHeight*4))

	// Combine the sections
	for i := 0; i < numSections; i++ {
		sectionPath := filepath.Join(outDir, "sprites", fmt.Sprintf("section_%d.png", i))
		sectionImg, err := loadImage(sectionPath)
		if err != nil {
			return fmt.Errorf("failed to open %s: %w", sectionPath, err)
		}

		// Copy the section into the combined image at the correct position
		destY := i * sectionHeight
		for y := 0; y < sectionHeight; y++ {
			for x := 0; x < sectionWidth; x++ {
				combined.Set(x, destY+y, sectionImg.At(x, y))
			}
		}
	}

	// Fill remaining sections with transparency
	for i := numSections; i < 4; i++ {
		destY := i * sectionHeight
		for y := 0; y < sectionHeight; y++ {
			for x := 0; x < sectionWidth; x++ {
				combined.Set(x, destY+y, color.RGBA{0, 0, 0, 0})
			}
		}
	}

	if err := saveAsPng(combined, filepath.Join(outDir, "spritesheet.png")); err != nil {
		return fmt.Errorf("failed to save spritesheet.png: %w", err)
	}

	return nil
}

// createIndividualSpritePNGs creates a PNG for each sprite described by
// outDir/spritesheet.json, writing them into outDir/sprites.
func createIndividualSpritePNGs(outDir string) error {
	data, err := os.ReadFile(filepath.Join(outDir, "spritesheet.json"))
	if err != nil {
		return fmt.Errorf("error reading JSON file: %w", err)
	}

	var spriteSheet SpriteSheet
	if err := json.Unmarshal(data, &spriteSheet); err != nil {
		return fmt.Errorf("error unmarshaling JSON: %w", err)
	}

	spritesDir := filepath.Join(outDir, "sprites")
	if err := os.MkdirAll(spritesDir, 0755); err != nil {
		return fmt.Errorf("error creating sprites directory: %w", err)
	}

	for _, sprite := range spriteSheet.Sprites {
		img := image.NewRGBA(image.Rect(0, 0, sprite.Width, sprite.Height))

		for y := 0; y < sprite.Height; y++ {
			for x := 0; x < sprite.Width; x++ {
				colorIndex := sprite.Pixels[y][x]
				if colorIndex >= 0 && colorIndex < len(spriteSheet.Metadata.Palette) {
					col := spriteSheet.Metadata.Palette[colorIndex]
					img.Set(x, y, color.RGBA{col.R, col.G, col.B, col.A})
				} else {
					img.Set(x, y, color.RGBA{0, 0, 0, 255})
				}
			}
		}

		filename := filepath.Join(spritesDir, sprite.Filename)
		if err := saveAsPng(img, filename); err != nil {
			return fmt.Errorf("error saving sprite %d: %w", sprite.ID, err)
		}
	}

	return nil
}
