package main //nolint:revive

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/josegonzalez/parsepico/pico8"
)

func main() {
	// Flags: user can specify a cart path, and optional --3 or --4
	var cartPath string
	var useSection3, useSection4 bool
	var cleanSlate bool

	flag.StringVar(&cartPath, "cart", "", "Path to the PICO-8 cartridge file (.p8)")
	flag.BoolVar(&useSection3, "3", false, "Include dual-purpose section 3 (sprites 128..191)")
	flag.BoolVar(&useSection4, "4", false, "Include dual-purpose section 4 (sprites 192..255)")
	flag.BoolVar(&cleanSlate, "clean", false, "Remove old sprites directory, map.png, spritesheet.png if they exist")
	flag.Parse()

	if cartPath == "" {
		fmt.Fprintln(os.Stderr, "Error: --cart flag is required")
		flag.Usage()
		os.Exit(1)
	}

	// Expand ~ and make the path absolute
	if strings.HasPrefix(cartPath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting user home directory: %v\n", err)
			os.Exit(1)
		}
		cartPath = filepath.Join(homeDir, cartPath[2:])
	}
	absPath, err := filepath.Abs(cartPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error making path absolute: %v\n", err)
		os.Exit(1)
	}

	opts := pico8.Options{
		UseSection3: useSection3,
		UseSection4: useSection4,
		Clean:       cleanSlate,
	}
	// The CLI writes outputs into the current working directory.
	if err := pico8.Extract(absPath, ".", opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Successfully exported cart data.")
}
