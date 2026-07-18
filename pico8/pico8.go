package pico8

import (
	"fmt"
	"os"
	"path/filepath"
)

// Output category names, valid values for Options.Only.
const (
	OutputMetadata    = "metadata"    // metadata.json
	OutputSpritesheet = "spritesheet" // spritesheet.png, spritesheet.json, sprites/section_*.png
	OutputSprites     = "sprites"     // sprites/sprite_*.png
	OutputMap         = "map"         // map.png, map.json
)

// Output filenames written into the target directory.
const (
	fileMetadataJSON    = "metadata.json"
	fileSpritesheetJSON = "spritesheet.json"
	fileSpritesheetPNG  = "spritesheet.png"
	fileMapPNG          = "map.png"
	fileMapJSON         = "map.json"
)

// Options controls how a cart is exported.
type Options struct {
	UseSection3 bool // include dual-purpose section 3 (sprites 128..191)
	UseSection4 bool // include dual-purpose section 4 (sprites 192..255)
	Clean       bool // remove old artifacts in outDir before exporting

	// Only limits which output categories are produced (OutputMetadata,
	// OutputSpritesheet, OutputSprites, OutputMap). An empty Only produces
	// every category.
	Only []string
}

// wants reports whether the named output category should be produced.
func (o Options) wants(name string) bool {
	if len(o.Only) == 0 {
		return true
	}
	for _, n := range o.Only {
		if n == name {
			return true
		}
	}
	return false
}

// Extract parses the PICO-8 .p8 cart at cartPath and writes the selected
// outputs into outDir: sprites/section_*.png and sprites/sprite_*.png,
// spritesheet.png, spritesheet.json, metadata.json, and (when the cart has a
// map) map.png and map.json. outDir is created if it does not exist. Use
// Options.Only to limit which categories are produced.
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

	// Potential dual-purpose sections. Each sprite row is 8 pixels: section 3
	// covers sprites 128..191 (rows 64..95), section 4 covers sprites 192..255
	// (rows 96..127).
	var dualPurposeSection1, dualPurposeSection2 []string
	if opts.UseSection3 {
		dualPurposeSection1 = dualPurposeRows(gfxData, 8*8)
	}
	if opts.UseSection4 {
		dualPurposeSection2 = dualPurposeRows(gfxData, 12*8)
	}

	// Full 16x16 sprite sheet image, needed for the sections and the map.
	spriteSheet := reconstructImage(gfxData)

	// metadata.json
	if opts.wants(OutputMetadata) {
		if err := saveMetadataJSON(generateMetadataJSON(cartName), filepath.Join(outDir, fileMetadataJSON)); err != nil {
			return fmt.Errorf("saving %s: %w", fileMetadataJSON, err)
		}
	}

	// The spritesheet and the individual sprites both need the sprite data.
	wantSheet := opts.wants(OutputSpritesheet)
	wantSprites := opts.wants(OutputSprites)
	if wantSheet || wantSprites {
		sheet, err := generateSpriteSheetJSON(gfxData, flagData, opts.UseSection3, opts.UseSection4)
		if err != nil {
			return fmt.Errorf("generating spritesheet JSON: %w", err)
		}
		if wantSheet {
			if err := saveSprites(spriteSheet, outDir, opts.UseSection3, opts.UseSection4); err != nil {
				return fmt.Errorf("saving sprites: %w", err)
			}
			if err := combineSectionsIntoSpriteSheet(outDir, availableSections(opts.UseSection3, opts.UseSection4)); err != nil {
				return fmt.Errorf("combining sections: %w", err)
			}
			if err := saveSpritesheetJSON(sheet, filepath.Join(outDir, fileSpritesheetJSON)); err != nil {
				return fmt.Errorf("saving %s: %w", fileSpritesheetJSON, err)
			}
		}
		if wantSprites {
			if err := createIndividualSpritePNGs(sheet, outDir); err != nil {
				return fmt.Errorf("creating individual sprite PNGs: %w", err)
			}
		}
	}

	// map.png + map.json (only when the cart has a map)
	if hasMapData && opts.wants(OutputMap) {
		mapImage := renderMap(mapData, dualPurposeSection1, dualPurposeSection2, spriteSheet, 128, mapHeight(opts))
		if err := saveAsPng(mapImage, filepath.Join(outDir, fileMapPNG)); err != nil {
			return fmt.Errorf("saving %s: %w", fileMapPNG, err)
		}
		mapSheet, err := generateMapJSON(mapData, gfxData, opts.UseSection3, opts.UseSection4)
		if err != nil {
			return fmt.Errorf("generating map JSON: %w", err)
		}
		if err := saveMapJSON(mapSheet, filepath.Join(outDir, fileMapJSON)); err != nil {
			return fmt.Errorf("saving %s: %w", fileMapJSON, err)
		}
	}

	return nil
}

// mapHeight returns the rendered map height in tiles for the given options.
func mapHeight(opts Options) int {
	h := 32
	if opts.UseSection3 {
		h = 48
	}
	if opts.UseSection4 {
		h = 64
	}
	return h
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
	for _, name := range []string{fileMapPNG, fileSpritesheetPNG, fileSpritesheetJSON, fileMetadataJSON, fileMapJSON} {
		_ = os.Remove(filepath.Join(outDir, name))
	}
}
