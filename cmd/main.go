package main

import (
	"fmt"

	"github.com/0dev1337/SpotifyDL/internal/helpers"
	"github.com/0dev1337/SpotifyDL/pkg/spotify"
)

func main() {
	helpers.CheckDependencies()

	client, err := spotify.NewClient()
	if err != nil {
		fmt.Printf("Error creating Spotify client: %v\n", err)
		return
	}
	if err := client.Setup(); err != nil {
		fmt.Printf("Error setting up Spotify client: %v\n", err)
		return
	}

}
