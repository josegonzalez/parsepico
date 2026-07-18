package pico8

import (
	"fmt"
	"os"
	"path/filepath"
)

// Options controls how a cart is exported.
type Options struct {
	UseSection3 bool // include dual-purpose section 3 (sprites 128..191)
	UseSection4 bool // include dual-purpose section 4 (sprites 192..255)
	Clean       bool // remove old artifacts in outDir before exporting
}

// Extract parses the PICO-8 .p8 cart at cartPath and writes all outputs into
// outDir: sprites/section_*.png and sprites/sprite_*.png, spritesheet.png,
// spritesheet.json, metadata.json, and (when the cart has a map) map.png and
// map.json. outDir is created if it does not exist.
func Extract(cartPath, outDir string, opts Options) error {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	if opts.Clean {
		removeOldArtifacts(outDir)
	}

	// Parse sections from the PICO-8 cart
	gfxData := parseSection(cartPath, "__gfx__")
	if len(gfxData) == 0 {
		return fmt.Errorf("no __gfx__ section found in cart %s", cartPath)
	}
	mapData := parseSection(cartPath, "__map__")
	hasMapData := len(mapData) > 0

	flagData := parseFlagSection(cartPath)
	cartName := parseCartName(cartPath)

	// Potential dual-purpose sections. Each sprite row is 8 pixels:
	// section 3 covers sprites 128..191 (rows 64..95), section 4 covers
	// sprites 192..255 (rows 96..127).
	var dualPurposeSection1, dualPurposeSection2 []string
	if opts.UseSection3 {
		dualPurposeSection1 = dualPurposeRows(gfxData, 8*8)
	}
	if opts.UseSection4 {
		dualPurposeSection2 = dualPurposeRows(gfxData, 12*8)
	}

	// Create full 16x16 sprite sheet
	spriteSheet := reconstructImage(gfxData)

	// Decide map height
	mapHeight := 32
	if opts.UseSection3 {
		mapHeight = 48
	}
	if opts.UseSection4 {
		mapHeight = 64
	}

	// Render map with optional dual-purpose sections only if map data exists
	if hasMapData {
		mapImage := renderMap(mapData, dualPurposeSection1, dualPurposeSection2, spriteSheet, 128, mapHeight)
		if err := saveAsPng(mapImage, filepath.Join(outDir, "map.png")); err != nil {
			return fmt.Errorf("saving map.png: %w", err)
		}
	}

	// Save sprite sub-sections, then combine them into a single sprite sheet
	if err := saveSprites(spriteSheet, outDir, opts.UseSection3, opts.UseSection4); err != nil {
		return fmt.Errorf("saving sprites: %w", err)
	}
	numSections := availableSections(opts.UseSection3, opts.UseSection4)
	if err := combineSectionsIntoSpriteSheet(outDir, numSections); err != nil {
		return fmt.Errorf("combining sections: %w", err)
	}

	// Generate and save spritesheet JSON
	jsonData, err := generateSpriteSheetJSON(gfxData, flagData, opts.UseSection3, opts.UseSection4)
	if err != nil {
		return fmt.Errorf("generating spritesheet JSON: %w", err)
	}
	if err := saveSpritesheetJSON(jsonData, filepath.Join(outDir, "spritesheet.json")); err != nil {
		return fmt.Errorf("saving spritesheet.json: %w", err)
	}

	// Generate and save cart metadata JSON
	if err := saveMetadataJSON(generateMetadataJSON(cartName), filepath.Join(outDir, "metadata.json")); err != nil {
		return fmt.Errorf("saving metadata.json: %w", err)
	}

	// Create individual sprite PNGs from the spritesheet JSON
	if err := createIndividualSpritePNGs(outDir); err != nil {
		return fmt.Errorf("creating individual sprite PNGs: %w", err)
	}

	// Generate and save map JSON only if map data exists
	if hasMapData {
		mapSheet, err := generateMapJSON(mapData, gfxData, opts.UseSection3, opts.UseSection4)
		if err != nil {
			return fmt.Errorf("generating map JSON: %w", err)
		}
		if err := saveMapJSON(mapSheet, filepath.Join(outDir, "map.json")); err != nil {
			return fmt.Errorf("saving map.json: %w", err)
		}
	}

	return nil
}

// dualPurposeRows returns the 32 gfx rows starting at startRow (clamped to the
// available data), or nil if startRow is out of range.
func dualPurposeRows(gfxData []string, startRow int) []string {
	endRow := startRow + 32
	if endRow > len(gfxData) {
		endRow = len(gfxData)
	}
	if startRow >= len(gfxData) {
		return nil
	}
	return gfxData[startRow:endRow]
}

// removeOldArtifacts deletes previously generated output files in outDir,
// ignoring any that don't exist.
func removeOldArtifacts(outDir string) {
	_ = os.RemoveAll(filepath.Join(outDir, "sprites"))
	for _, name := range []string{"map.png", "spritesheet.png", "spritesheet.json", "metadata.json", "map.json"} {
		_ = os.Remove(filepath.Join(outDir, name))
	}
}
