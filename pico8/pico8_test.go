package pico8

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	// titleCartFile is used where the filename is irrelevant because the title
	// comes from the __lua__ header.
	titleCartFile = "whatever.p8"
	// fallbackFile / fallbackName exercise the filename fallback.
	fallbackFile = "mygame.p8"
	fallbackName = "mygame"
	celesteTitle = "celeste"
)

// writeTempCart writes cart content to a file named name inside a temp dir and
// returns its path.
func writeTempCart(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing temp cart: %v", err)
	}
	return path
}

func TestParseCartName(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		content string
		want    string
	}{
		{"title comment", titleCartFile, "__lua__\n-- celeste\n-- by matt\n__gfx__\n0\n", celesteTitle},
		{"one leading space only", titleCartFile, "__lua__\n--  celeste\n", " celeste"},
		{"decoration preserved", titleCartFile, "__lua__\n-- ~celeste~\n", "~celeste~"},
		{"author line ignored", titleCartFile, "__lua__\n-- my game\n-- by jose\n", "my game"},
		{"crlf tolerated", titleCartFile, "__lua__\r\n-- celeste\r\n", celesteTitle},
		{"fallback when first line is code", fallbackFile, "__lua__\nfunction _init() end\n", fallbackName},
		{"fallback when no lua section", fallbackFile, "__gfx__\n00700700\n", fallbackName},
		{"fallback when comment is empty", fallbackFile, "__lua__\n--\n", fallbackName},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTempCart(t, tt.file, tt.content)
			if got := parseCartName(path); got != tt.want {
				t.Errorf("parseCartName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGenerateMetadataJSON(t *testing.T) {
	if m := generateMetadataJSON(celesteTitle); m.CartName != celesteTitle {
		t.Errorf("CartName = %q, want %q", m.CartName, celesteTitle)
	}
}

// sampleCartPath writes a minimal cart (titled "test cart", 8 gfx rows) and
// returns its path.
func sampleCartPath(t *testing.T) string {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("pico-8 cartridge // http://www.pico-8.com\n")
	sb.WriteString("version 42\n")
	sb.WriteString("__lua__\n-- test cart\n-- by me\nfunction _init() end\n")
	sb.WriteString("__gfx__\n")
	gfxLine := strings.Repeat("7", 128)
	for i := 0; i < 8; i++ {
		sb.WriteString(gfxLine + "\n")
	}
	return writeTempCart(t, "mycart.p8", sb.String())
}

func TestExtract(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "out")
	if err := Extract(sampleCartPath(t), outDir, Options{}); err != nil {
		t.Fatalf("Extract: %v", err)
	}

	// metadata.json carries the cart name from the __lua__ header.
	data, err := os.ReadFile(filepath.Join(outDir, fileMetadataJSON))
	if err != nil {
		t.Fatalf("reading metadata.json: %v", err)
	}
	var meta CartMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("unmarshaling metadata.json: %v", err)
	}
	if meta.CartName != "test cart" {
		t.Errorf("cartName = %q, want %q", meta.CartName, "test cart")
	}

	// The expected output files were produced.
	for _, f := range []string{fileSpritesheetJSON, fileSpritesheetPNG, fileMetadataJSON} {
		if _, err := os.Stat(filepath.Join(outDir, f)); err != nil {
			t.Errorf("missing %s: %v", f, err)
		}
	}
	if _, err := os.Stat(filepath.Join(outDir, "sprites", "sprite_000.png")); err != nil {
		t.Errorf("missing sprites/sprite_000.png: %v", err)
	}
}

func TestExtractOnly(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "out")
	if err := Extract(sampleCartPath(t), outDir, Options{Only: []string{OutputMetadata}}); err != nil {
		t.Fatalf("Extract: %v", err)
	}

	// The selected category is produced.
	if _, err := os.Stat(filepath.Join(outDir, fileMetadataJSON)); err != nil {
		t.Errorf("metadata.json missing: %v", err)
	}
	// Unselected categories are not produced at all.
	for _, f := range []string{fileSpritesheetJSON, fileSpritesheetPNG, "sprites"} {
		if _, err := os.Stat(filepath.Join(outDir, f)); !os.IsNotExist(err) {
			t.Errorf("%s should not exist (err=%v)", f, err)
		}
	}
}
