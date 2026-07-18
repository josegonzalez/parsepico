package pico8

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// parseSection reads lines between a given marker (e.g. __gfx__) until next marker __*
func parseSection(filePath, sectionName string) []string {
	f, err := os.Open(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open cart file: %v\n", err)
		return nil
	}
	defer f.Close() //nolint:errcheck

	var section []string
	inSection := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		// If we encounter the marker (e.g. __gfx__), we start capturing
		if strings.HasPrefix(line, sectionName) {
			inSection = true
			continue
		}
		// If we see any other marker (e.g. __lua__, __map__, etc.) we stop
		if strings.HasPrefix(line, "__") && line != sectionName {
			inSection = false
		}

		if inSection {
			section = append(section, line)
		}
	}
	return section
}

// parseFlagSection reads the __gff__ section and returns the flag data for each sprite
func parseFlagSection(filePath string) []int {
	flagData := make([]int, 256) // Initialize with 0s

	section := parseSection(filePath, "__gff__")
	if len(section) == 0 {
		return flagData // Return all zeros if no flag data found
	}

	// Each line in __gff__ contains 256 hex chars (2 per sprite, 128 sprites per line)
	// We need 2 lines to cover all 256 sprites
	for lineNum, line := range section {
		if lineNum >= 2 { // We only need first 2 lines
			break
		}

		// Process each pair of hex chars
		for i := 0; i < len(line)-1 && i/2 < 128; i += 2 {
			spriteIndex := (lineNum * 128) + (i / 2)
			if spriteIndex >= 256 {
				break
			}

			// Convert two hex chars to a byte
			flagValue := parseHexChar(rune(line[i]))*16 + parseHexChar(rune(line[i+1]))
			flagData[spriteIndex] = flagValue
		}
	}

	return flagData
}

// parseCartName derives the cart name. Mirrors fake-08: the name is the first
// line of the __lua__ section IF that line is a Lua comment (-- title), with
// the leading "--" and one optional space stripped. Decorations like ~title~
// are left intact and the "-- by author" line is not included. When there is
// no usable title comment (no __lua__ section, first line is code, or the
// comment is empty), it falls back to the cart's filename without extension.
func parseCartName(filePath string) string {
	lua := parseSection(filePath, "__lua__")
	if len(lua) > 0 {
		first := strings.TrimRight(lua[0], " \r\n")
		if strings.HasPrefix(first, "--") {
			if title := strings.TrimPrefix(first[2:], " "); title != "" {
				return title
			}
		}
	}
	// Fallback: e.g. /path/celeste.p8 -> "celeste".
	base := filepath.Base(filePath)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// parseHexChar interprets a single hex digit (0..F)
func parseHexChar(c rune) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}

// getFlagArray converts a flag byte into array of 8 booleans
func getFlagArray(flagByte int) []bool {
	flags := make([]bool, 8)
	for i := 0; i < 8; i++ {
		flags[i] = (flagByte & (1 << i)) != 0
	}
	return flags
}
