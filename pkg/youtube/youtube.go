package youtube

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/0dev1337/SpotifyDL/internal/tools"
	"github.com/0dev1337/SpotifyDL/pkg/spotify"
)

const (
	defaultOutputDir   = "downloads"
	downloadTimeout    = 30 * time.Minute
)

func DownloadMusic(track spotify.TrackData) error {
	paths, err := tools.Resolve("", "")
	if err != nil {
		return fmt.Errorf("resolve tools: %w", err)
	}

	if err := os.MkdirAll(defaultOutputDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	query := searchQuery(track)
	outputTemplate := filepath.Join(defaultOutputDir, sanitizeFilename(baseFilename(track))+".%(ext)s")

	args := []string{
		"--ffmpeg-location", paths.FFmpeg,
		"--no-playlist",
		"--quiet",
		"--no-progress",
		"--remote-components", "ejs:github",
		"--extractor-args", "youtube:player_client=default,-android_sdkless",
		"-f", "bestaudio/best",
		"-x",
		"--audio-format", "mp3",
		"--audio-quality", "0",
		"-o", outputTemplate,
		"ytsearch1:" + query,
	}
	args = appendJSRuntimeArgs(args)

	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, paths.YTDLP, args...)
	cmd.Stderr = io.Discard

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("download failed for %q", track.Name)
	}

	if err := embedMetadata(outputPath(track), track); err != nil {
		return fmt.Errorf("embed metadata for %q: %w", track.Name, err)
	}

	return nil
}

func appendJSRuntimeArgs(args []string) []string {
	for _, runtime := range []string{"deno", "node"} {
		if path, err := exec.LookPath(runtime); err == nil {
			return append(args, "--js-runtimes", runtime+":"+path)
		}
	}
	return args
}

func searchQuery(track spotify.TrackData) string {
	artists := strings.Join(track.ArtistNames(), " ")
	if artists == "" {
		return track.Name
	}
	return artists + " - " + track.Name
}

func baseFilename(track spotify.TrackData) string {
	return searchQuery(track)
}

func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		`\`, "_",
		`/`, "_",
		`:`, "_",
		`*`, "_",
		`?`, "_",
		`"`, "_",
		`<`, "_",
		`>`, "_",
		`|`, "_",
	)
	return strings.TrimSpace(replacer.Replace(name))
}
