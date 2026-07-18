package pico8

import (
	"encoding/json"
	"fmt"
	"os"
)

// generateSpriteSheetJSON creates the JSON representation of the spritesheet
func generateSpriteSheetJSON(gfxData []string, flagData []int, useSection3, useSection4 bool) (*SpriteSheet, error) {
	spriteSheet := &SpriteSheet{
		Version:     jsonVersion,
		Description: "PICO-8 spritesheet export",
		Sprites:     make([]Sprite, 0),
		Metadata: MetaData{
			SpriteWidth:  8,
			SpriteHeight: 8,
			GridWidth:    16,
			GridHeight:   16,
			AvailableSprites: AvailableSprites{
				Total: 128, // Default to base sprites only
				Ranges: []SpriteRange{
					{
						Start:       0,
						End:         127,
						Used:        true,
						Description: "Base sprites",
					},
				},
				Sections: SpriteSections{
					Base:     true,
					Section3: useSection3,
					Section4: useSection4,
				},
			},
			Palette: make([]PaletteColor, len(pico8Palette)),
		},
	}

	// Convert palette to JSON format
	for i, col := range pico8Palette {
		spriteSheet.Metadata.Palette[i] = PaletteColor{
			R: col.R,
			G: col.G,
			B: col.B,
			A: col.A,
		}
	}

	// Update available sprites based on sections
	if !useSection3 {
		spriteSheet.Metadata.AvailableSprites.Ranges = append(
			spriteSheet.Metadata.AvailableSprites.Ranges,
			SpriteRange{
				Start:       128,
				End:         191,
				Used:        true,
				Description: "Section 3 sprites",
			},
		)
		spriteSheet.Metadata.AvailableSprites.Total += 64
	}
	if !useSection4 {
		spriteSheet.Metadata.AvailableSprites.Ranges = append(
			spriteSheet.Metadata.AvailableSprites.Ranges,
			SpriteRange{
				Start:       192,
				End:         255,
				Used:        true,
				Description: "Section 4 sprites",
			},
		)
		spriteSheet.Metadata.AvailableSprites.Total += 64
	}

	// Process each sprite, but only include those in available ranges
	for spriteID := 0; spriteID < 256; spriteID++ {
		// Skip if sprite is in an unused section
		if (spriteID >= 128 && spriteID < 192 && useSection3) ||
			(spriteID >= 192 && useSection4) {
			continue
		}

		x := (spriteID % 16)
		y := (spriteID / 16)

		// Create pixel data for this sprite
		pixels := make([][]int, 8)
		for i := range pixels {
			pixels[i] = make([]int, 8)
			if y*8+i < len(gfxData) {
				line := gfxData[y*8+i]
				for j := 0; j < 8 && x*8+j < len(line); j++ {
					if x*8+j < len(line) {
						pixels[i][j] = parseHexChar(rune(line[x*8+j]))
					}
				}
			}
		}

		// Check if sprite is used (not blank)
		used := false
		for _, row := range pixels {
			for _, pixel := range row {
				if pixel != 0 {
					used = true
					break
				}
			}
			if used {
				break
			}
		}

		// Create sprite entry
		sprite := Sprite{
			ID:       spriteID,
			X:        x,
			Y:        y,
			Width:    8,
			Height:   8,
			Pixels:   pixels,
			Flags:    SpriteFlags{Bitfield: flagData[spriteID], Individual: getFlagArray(flagData[spriteID])},
			Used:     used,
			Filename: fmt.Sprintf("sprite_%03d.png", spriteID),
		}

		// Only include sprites that have at least one non-zero pixel
		if used {
			spriteSheet.Sprites = append(spriteSheet.Sprites, sprite)
		}
	}

	return spriteSheet, nil
}

// generateMapJSON creates the JSON representation of the map
func generateMapJSON(mapData, gfxData []string, useSection3, useSection4 bool) (*MapSheet, error) {
	// Calculate map dimensions
	width := 128 // Default width
	height := 32 // Default height
	if useSection3 {
		height = 48
	}
	if useSection4 {
		height = 64
	}

	mapSheet := &MapSheet{
		Version:     jsonVersion,
		Description: "PICO-8 map export",
		Width:       width,
		Height:      height,
		Name:        "main",
		Cells:       make([]MapCell, 0),
	}

	// Process main map layer (base section)
	for y := 0; y < 32; y++ {
		if y < len(mapData) {
			line := mapData[y]
			for x := 0; x < width; x++ {
				if x*2+1 < len(line) {
					// Each tile is represented by 2 hex digits
					spriteY := parseHexChar(rune(line[x*2]))
					spriteX := parseHexChar(rune(line[x*2+1]))
					// Convert sprite coordinates to sprite ID
					spriteID := spriteY*16 + spriteX

					// Skip cells with sprite ID 0
					if spriteID != 0 {
						cell := MapCell{
							X:      x,
							Y:      y,
							Sprite: spriteID,
						}
						mapSheet.Cells = append(mapSheet.Cells, cell)
					}
				}
			}
		}
	}

	// Process section 3 if enabled (maps to rows 32–47)
	if useSection3 {
		startRow := 8 * 8       // 64
		endRow := startRow + 32 // 96
		if endRow > len(gfxData) {
			endRow = len(gfxData)
		}
		if startRow < len(gfxData) {
			section3Data := gfxData[startRow:endRow]
			for y := 0; y < len(section3Data); y++ {
				line := section3Data[y]
				yIsEven := (y % 2) == 0
				// Iterate over half the width of the line (64 tiles per row)
				for x := 0; x < len(line)/2; x++ {
					// First hex digit is spriteX, second is spriteY
					spriteX := parseHexChar(rune(line[x*2]))
					spriteY := parseHexChar(rune(line[x*2+1]))
					// Convert sprite coordinates to sprite ID (0-127)
					spriteID := spriteY*16 + spriteX

					if yIsEven {
						// Left half of the map (rows 32-47)
						// Skip cells with sprite ID 0
						if spriteID != 0 {
							cell := MapCell{
								X:      x,
								Y:      32 + (y / 2),
								Sprite: spriteID,
							}
							mapSheet.Cells = append(mapSheet.Cells, cell)
						}
					} else {
						// Right half of the map (rows 32-47)
						// Skip cells with sprite ID 0
						if spriteID != 0 {
							cell := MapCell{
								X:      64 + x,
								Y:      32 + ((y - 1) / 2),
								Sprite: spriteID,
							}
							mapSheet.Cells = append(mapSheet.Cells, cell)
						}
					}
				}
			}
		}
	}

	// Process section 4 if enabled (maps to rows 48–63)
	if useSection4 {
		startRow := 12 * 8      // 96
		endRow := startRow + 32 // 128
		if endRow > len(gfxData) {
			endRow = len(gfxData)
		}
		if startRow < len(gfxData) {
			section4Data := gfxData[startRow:endRow]
			for y := 0; y < len(section4Data); y++ {
				line := section4Data[y]
				yIsEven := (y % 2) == 0
				// Iterate over half the width of the line (64 tiles per row)
				for x := 0; x < len(line)/2; x++ {
					// First hex digit is spriteX, second is spriteY
					spriteX := parseHexChar(rune(line[x*2]))
					spriteY := parseHexChar(rune(line[x*2+1]))
					// Convert sprite coordinates to sprite ID (0-127)
					spriteID := spriteY*16 + spriteX

					if yIsEven {
						// Left half of the map (rows 48-63)
						// Skip cells with sprite ID 0
						if spriteID != 0 {
							cell := MapCell{
								X:      x,
								Y:      48 + (y / 2),
								Sprite: spriteID,
							}
							mapSheet.Cells = append(mapSheet.Cells, cell)
						}
					} else {
						// Right half of the map (rows 48-63)
						// Skip cells with sprite ID 0
						if spriteID != 0 {
							cell := MapCell{
								X:      64 + x,
								Y:      48 + ((y - 1) / 2),
								Sprite: spriteID,
							}
							mapSheet.Cells = append(mapSheet.Cells, cell)
						}
					}
				}
			}
		}
	}

	return mapSheet, nil
}

// generateMetadataJSON builds the cart-level metadata written to metadata.json.
func generateMetadataJSON(cartName string) *CartMetadata {
	return &CartMetadata{
		Version:     jsonVersion,
		Description: "PICO-8 cart metadata",
		CartName:    cartName,
	}
}

// saveSpritesheetJSON saves the spritesheet data as JSON
func saveSpritesheetJSON(spriteSheet *SpriteSheet, path string) error {
	data, err := json.MarshalIndent(spriteSheet, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// saveMapJSON saves the map data as JSON
func saveMapJSON(mapSheet *MapSheet, path string) error {
	data, err := json.MarshalIndent(mapSheet, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling map JSON: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// saveMetadataJSON saves the cart metadata as JSON
func saveMetadataJSON(metadata *CartMetadata, path string) error {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling metadata JSON: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}
