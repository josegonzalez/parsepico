// Package pico8 parses PICO-8 .p8 text cartridges and exports their sprites,
// map, and metadata as PNG images and JSON files.
package pico8

import "image/color"

// jsonVersion is the schema version stamped on every JSON output file.
const jsonVersion = "1.0"

// pico8Palette is the PICO-8 16-color palette.
var pico8Palette = []color.RGBA{
	{0, 0, 0, 255},       // 0: Black
	{29, 43, 83, 255},    // 1: Dark Blue
	{126, 37, 83, 255},   // 2: Dark Purple
	{0, 135, 81, 255},    // 3: Dark Green
	{171, 82, 54, 255},   // 4: Brown
	{95, 87, 79, 255},    // 5: Dark Gray
	{194, 195, 199, 255}, // 6: Light Gray
	{255, 241, 232, 255}, // 7: White
	{255, 0, 77, 255},    // 8: Red
	{255, 163, 0, 255},   // 9: Orange
	{255, 236, 39, 255},  // 10: Yellow
	{0, 228, 54, 255},    // 11: Green
	{41, 173, 255, 255},  // 12: Blue
	{131, 118, 156, 255}, // 13: Indigo
	{255, 119, 168, 255}, // 14: Pink
	{255, 204, 170, 255}, // 15: Peach
}

// SpriteSheet represents the complete spritesheet data for JSON output.
type SpriteSheet struct {
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Sprites     []Sprite `json:"sprites"`
	Metadata    MetaData `json:"metadata"`
}

// Sprite represents a single sprite with pixel data and flags.
type Sprite struct {
	ID       int         `json:"id"`
	X        int         `json:"x"`
	Y        int         `json:"y"`
	Width    int         `json:"width"`
	Height   int         `json:"height"`
	Pixels   [][]int     `json:"pixels"`
	Flags    SpriteFlags `json:"flags"`
	Used     bool        `json:"used"`
	Filename string      `json:"filename"`
}

// SpriteFlags contains flag data for a sprite.
type SpriteFlags struct {
	Bitfield   int    `json:"bitfield"`
	Individual []bool `json:"individual"`
}

// MetaData provides metadata information for the spritesheet.
type MetaData struct {
	SpriteWidth      int              `json:"spriteWidth"`
	SpriteHeight     int              `json:"spriteHeight"`
	GridWidth        int              `json:"gridWidth"`
	GridHeight       int              `json:"gridHeight"`
	AvailableSprites AvailableSprites `json:"availableSprites"`
	Palette          []PaletteColor   `json:"palette"`
}

// AvailableSprites contains information about available sprite ranges and sections.
type AvailableSprites struct {
	Total    int            `json:"total"`
	Ranges   []SpriteRange  `json:"ranges"`
	Sections SpriteSections `json:"sections"`
}

// SpriteRange defines a range of sprites by their start and end indices.
type SpriteRange struct {
	Start       int    `json:"start"`
	End         int    `json:"end"`
	Used        bool   `json:"used"`
	Description string `json:"description"`
}

// SpriteSections indicates which sprite sections are available.
type SpriteSections struct {
	Base     bool `json:"base"`
	Section3 bool `json:"section3"`
	Section4 bool `json:"section4"`
}

// PaletteColor represents a single color in the sprite palette.
type PaletteColor struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
	A uint8 `json:"a"`
}

// MapSheet represents the complete map data for JSON output.
type MapSheet struct {
	Version     string    `json:"version"`
	Description string    `json:"description"`
	Width       int       `json:"width"`
	Height      int       `json:"height"`
	Name        string    `json:"name"`
	Cells       []MapCell `json:"cells"`
}

// MapCell represents a single cell in the map.
type MapCell struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Sprite int `json:"sprite"`
}

// CartMetadata holds cart-level metadata written to metadata.json.
type CartMetadata struct {
	Version     string `json:"version"`
	Description string `json:"description"`
	CartName    string `json:"cartName"`
}
