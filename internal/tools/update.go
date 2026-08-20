package tools

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const ytdlpUpdateInterval = 24 * time.Hour

func isBundledYTDLP(path string) bool {
	platform, err := InstallPlatform()
	if err != nil {
		return false
	}
	marker := filepath.Join("tools", platform, toolFileName("yt-dlp"))
	return strings.HasSuffix(filepath.Clean(path), marker)
}

func updateStampPath(ytdlpPath string) string {
	return filepath.Join(filepath.Dir(ytdlpPath), ".yt-dlp-updated")
}

func updatedRecently(ytdlpPath string) bool {
	info, err := os.Stat(updateStampPath(ytdlpPath))
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) < ytdlpUpdateInterval
}

func markUpdated(ytdlpPath string) {
	_ = os.WriteFile(updateStampPath(ytdlpPath), []byte(time.Now().Format(time.RFC3339)), 0o644)
}

func UpdateBundledYTDLP(ctx context.Context, ytdlpPath string, onProgress ProgressFunc) {
	if !isBundledYTDLP(ytdlpPath) || updatedRecently(ytdlpPath) {
		return
	}

	if onProgress != nil {
		onProgress("Updating yt-dlp...")
	}

	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	_ = withYTDLPSetupLock(ctx, defaultInstallRoot(), func() error {
		cmd := exec.CommandContext(ctx, ytdlpPath, "-U")
		if err := cmd.Run(); err == nil {
			markUpdated(ytdlpPath)
		}
		return nil
	})
}
