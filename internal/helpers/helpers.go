package helpers

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/0dev1337/SpotifyDL/internal/tools"
)

func CheckDependencies() {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
	defer cancel()

	paths, err := tools.ResolveOrInstall(ctx, "", "")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("ffmpeg: %s\n", paths.FFmpeg)
	fmt.Printf("yt-dlp: %s\n", paths.YTDLP)
}
