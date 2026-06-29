package tools

import (
	"fmt"
	"path/filepath"
	"runtime"
)

func InstallPlatform() (string, error) {
	switch runtime.GOOS {
	case "windows":
		switch runtime.GOARCH {
		case "amd64", "386":
			return "windows-amd64", nil
		}
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			return "linux-amd64", nil
		case "arm64":
			return "linux-arm64", nil
		}
	case "darwin":
		switch runtime.GOARCH {
		case "amd64":
			return "darwin-amd64", nil
		case "arm64":
			return "darwin-arm64", nil
		}
	}

	return "", fmt.Errorf("unsupported platform %s/%s", runtime.GOOS, runtime.GOARCH)
}

func PlatformSearchDirs() []string {
	primary := runtime.GOOS + "-" + runtime.GOARCH
	dirs := []string{primary}

	// 32-bit windows builds still ship amd64 binaries
	if runtime.GOOS == "windows" && runtime.GOARCH == "386" {
		dirs = append(dirs, "windows-amd64")
	}

	return dirs
}

func ToolsDir(root string) (string, error) {
	platform, err := InstallPlatform()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "tools", platform), nil
}

func FFmpegDir(root string) (string, error) {
	toolsDir, err := ToolsDir(root)
	if err != nil {
		return "", err
	}
	return filepath.Join(toolsDir, "ffmpeg"), nil
}

func YTDLPPath(root string) (string, error) {
	toolsDir, err := ToolsDir(root)
	if err != nil {
		return "", err
	}
	return filepath.Join(toolsDir, toolFileName("yt-dlp")), nil
}
