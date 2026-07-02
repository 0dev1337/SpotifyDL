package main

import (
	"fmt"
	"os"

	"github.com/0dev1337/SpotifyDL/internal/tui"
)

func main() {
	if err := tui.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
