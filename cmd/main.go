package main

import (
	"context"
	"fmt"

	"github.com/0dev1337/SpotifyDL/internal/helpers"
	"github.com/0dev1337/SpotifyDL/pkg/spotify"
)

func main() {
	helpers.CheckDependencies()
	result, err := spotify.BuildTotp(context.Background(), 0, "")
	if err != nil {
		panic(err)
	}
	fmt.Printf("TOTP: %s\n", result.Totp)
	fmt.Printf("TOTP Server: %s\n", result.TotpServer)
	fmt.Printf("Client Time: %d\n", result.ClientTime)
	fmt.Printf("Server Time: %d\n", *result.ServerTime)
	fmt.Printf("Cipher: %s\n", result.Cipher)
	fmt.Printf("Version: %d\n", result.Version)

}
