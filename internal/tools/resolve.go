package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const (
	envFFmpeg = "FFMPEG_PATH"
	envYTDLP  = "YTDLP_PATH"
)

type Paths struct {
	FFmpeg string
	YTDLP  string
}

// flag → env → tools/ → PATH
func Resolve(ffmpegPath, ytdlpPath string) (Paths, error) {
	ffmpeg, err := resolveTool(ffmpegPath, envFFmpeg, "ffmpeg")
	if err != nil {
		return Paths{}, fmt.Errorf("ffmpeg: %w", err)
	}

	ytdlp, err := resolveTool(ytdlpPath, envYTDLP, "yt-dlp")
	if err != nil {
		return Paths{}, fmt.Errorf("yt-dlp: %w", err)
	}

	return Paths{FFmpeg: ffmpeg, YTDLP: ytdlp}, nil
}

// Same as Resolve, but pulls down tools/ first when nothing is configured.
func ResolveOrInstall(ctx context.Context, ffmpegPath, ytdlpPath string) (Paths, error) {
	return ResolveOrInstallWithProgress(ctx, ffmpegPath, ytdlpPath, nil)
}

func ResolveOrInstallWithProgress(ctx context.Context, ffmpegPath, ytdlpPath string, onProgress ProgressFunc) (Paths, error) {
	paths, err := Resolve(ffmpegPath, ytdlpPath)
	if err == nil {
		UpdateBundledYTDLP(ctx, paths.YTDLP, onProgress)
		return paths, nil
	}

	if !shouldAutoInstall(ffmpegPath, ytdlpPath) {
		return Paths{}, err
	}

	resolveErr := err
	if installErr := Install(ctx, InstallOptions{OnProgress: onProgress}); installErr != nil {
		return Paths{}, fmt.Errorf("%w; auto-install failed: %v", resolveErr, installErr)
	}

	paths, err = Resolve(ffmpegPath, ytdlpPath)
	if err != nil {
		return Paths{}, fmt.Errorf("%w; still missing after auto-install: %v", resolveErr, err)
	}

	UpdateBundledYTDLP(ctx, paths.YTDLP, onProgress)
	return paths, nil
}

func shouldAutoInstall(ffmpegPath, ytdlpPath string) bool {
	if ffmpegPath != "" || ytdlpPath != "" {
		return false
	}
	if os.Getenv(envFFmpeg) != "" || os.Getenv(envYTDLP) != "" {
		return false
	}
	return true
}

func resolveTool(explicit, envKey, name string) (string, error) {
	if explicit != "" {
		return validateExecutable(explicit)
	}

	if v := os.Getenv(envKey); v != "" {
		return validateExecutable(v)
	}

	if bundled := bundledPath(name); bundled != "" {
		if path, err := validateExecutable(bundled); err == nil {
			return path, nil
		}
	}

	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("%s not found", name)
}

func validateExecutable(path string) (string, error) {
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s is a directory", path)
	}
	return path, nil
}

func bundledPath(name string) string {
	for _, root := range searchRoots() {
		switch name {
		case "ffmpeg":
			if path := bundledFFmpegPath(root); path != "" {
				return path
			}
		case "yt-dlp":
			if path := bundledYTDLPPath(root); path != "" {
				return path
			}
		}
	}

	return ""
}

func bundledFFmpegPath(root string) string {
	dir, err := FFmpegDir(root)
	if err != nil {
		return ""
	}

	candidates := []string{
		filepath.Join(dir, toolFileName("ffmpeg")),
		filepath.Join(dir, "bin", toolFileName("ffmpeg")),
	}
	for _, candidate := range candidates {
		if fileExists(candidate) {
			return candidate
		}
	}

	return ""
}

func bundledYTDLPPath(root string) string {
	path, err := YTDLPPath(root)
	if err != nil {
		return ""
	}
	if fileExists(path) {
		return path
	}
	return ""
}

func toolFileName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func searchRoots() []string {
	var roots []string

	if exe, err := os.Executable(); err == nil {
		exe, _ = filepath.EvalSymlinks(exe)
		roots = append(roots, filepath.Dir(exe))
	}

	if wd, err := os.Getwd(); err == nil {
		roots = append(roots, wd)
		if moduleRoot := findModuleRoot(wd); moduleRoot != "" {
			roots = append(roots, moduleRoot)
		}
	}

	return dedupe(roots)
}

func findModuleRoot(start string) string {
	dir := start
	for {
		if fileExists(filepath.Join(dir, "go.mod")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dedupe(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = filepath.Clean(item)
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
