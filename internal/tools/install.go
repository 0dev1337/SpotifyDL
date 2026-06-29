package tools

import (
	"archive/tar"
	"archive/zip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/ulikunitz/xz"
)

const (
	ffmpegReleaseBase = "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest"
	shakaReleaseBase  = "https://github.com/shaka-project/static-ffmpeg-binaries/releases/latest/download"
	ytdlpReleaseBase  = "https://github.com/yt-dlp/yt-dlp/releases/latest/download"
)

type installAssets struct {
	ffmpegURL    string
	ffmpegFormat string
	ytdlpURL     string
}

type InstallOptions struct {
	Root  string
	Force bool
}

func Install(ctx context.Context, opts InstallOptions) error {
	root, err := resolveInstallRoot(opts.Root)
	if err != nil {
		return err
	}

	assets, err := assetsForPlatform()
	if err != nil {
		return err
	}

	toolsDir, err := ToolsDir(root)
	if err != nil {
		return err
	}
	ffmpegDir, err := FFmpegDir(root)
	if err != nil {
		return err
	}
	ytdlpPath, err := YTDLPPath(root)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(toolsDir, 0o755); err != nil {
		return fmt.Errorf("create tools dir: %w", err)
	}

	if err := installFFmpeg(ctx, assets, ffmpegDir, opts.Force); err != nil {
		return fmt.Errorf("install ffmpeg: %w", err)
	}
	if err := installYTDLP(ctx, assets, ytdlpPath, opts.Force); err != nil {
		return fmt.Errorf("install yt-dlp: %w", err)
	}

	fmt.Printf("Done. Tools are in %s\n", toolsDir)
	return nil
}

func resolveInstallRoot(root string) (string, error) {
	if root != "" {
		return filepath.Clean(root), nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	if moduleRoot := findModuleRoot(wd); moduleRoot != "" {
		return moduleRoot, nil
	}

	return wd, nil
}

func assetsForPlatform() (installAssets, error) {
	platform, err := InstallPlatform()
	if err != nil {
		return installAssets{}, err
	}

	switch platform {
	case "windows-amd64":
		return installAssets{
			ffmpegURL:    ffmpegReleaseBase + "/ffmpeg-master-latest-win64-gpl-shared.zip",
			ffmpegFormat: "zip",
			ytdlpURL:     ytdlpReleaseBase + "/yt-dlp.exe",
		}, nil
	case "linux-amd64":
		return installAssets{
			ffmpegURL:    ffmpegReleaseBase + "/ffmpeg-master-latest-linux64-gpl-shared.tar.xz",
			ffmpegFormat: "tar.xz",
			ytdlpURL:     ytdlpReleaseBase + "/yt-dlp",
		}, nil
	case "linux-arm64":
		return installAssets{
			ffmpegURL:    ffmpegReleaseBase + "/ffmpeg-master-latest-linuxarm64-gpl-shared.tar.xz",
			ffmpegFormat: "tar.xz",
			ytdlpURL:     ytdlpReleaseBase + "/yt-dlp",
		}, nil
	case "darwin-amd64":
		return installAssets{
			ffmpegURL:    shakaReleaseBase + "/ffmpeg-osx-x64",
			ffmpegFormat: "binary",
			ytdlpURL:     ytdlpReleaseBase + "/yt-dlp_macos",
		}, nil
	case "darwin-arm64":
		return installAssets{
			ffmpegURL:    shakaReleaseBase + "/ffmpeg-osx-arm64",
			ffmpegFormat: "binary",
			ytdlpURL:     ytdlpReleaseBase + "/yt-dlp_macos",
		}, nil
	default:
		return installAssets{}, fmt.Errorf("unsupported install platform %s", platform)
	}
}

func installFFmpeg(ctx context.Context, assets installAssets, destDir string, force bool) error {
	ffmpegBin := filepath.Join(destDir, toolFileName("ffmpeg"))
	if !force && fileExists(ffmpegBin) {
		fmt.Printf("ffmpeg already present at %s\n", ffmpegBin)
		return nil
	}

	fmt.Printf("Downloading ffmpeg from %s\n", assets.ffmpegURL)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp("", "spotifydl-ffmpeg-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if err := downloadFile(ctx, assets.ffmpegURL, tmpFile); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	if assets.ffmpegFormat == "binary" {
		if err := copyFile(tmpPath, ffmpegBin); err != nil {
			return err
		}
		return os.Chmod(ffmpegBin, 0o755)
	}

	extractDir, err := os.MkdirTemp("", "spotifydl-ffmpeg-extract-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(extractDir)

	switch assets.ffmpegFormat {
	case "zip":
		if err := extractZip(tmpPath, extractDir); err != nil {
			return err
		}
	case "tar.xz":
		if err := extractTarXZ(tmpPath, extractDir); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported ffmpeg archive format %q", assets.ffmpegFormat)
	}

	bundleRoot, err := findFFmpegBundleRoot(extractDir)
	if err != nil {
		return err
	}

	if force {
		if err := os.RemoveAll(destDir); err != nil {
			return err
		}
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return err
		}
	}

	if runtime.GOOS == "windows" {
		return copyDir(filepath.Join(bundleRoot, "bin"), destDir)
	}

	// shared builds on unix need lib/ beside bin/
	if err := copyDir(filepath.Join(bundleRoot, "bin"), filepath.Join(destDir, "bin")); err != nil {
		return err
	}
	if dirExists(filepath.Join(bundleRoot, "lib")) {
		return copyDir(filepath.Join(bundleRoot, "lib"), filepath.Join(destDir, "lib"))
	}

	return nil
}

func installYTDLP(ctx context.Context, assets installAssets, destPath string, force bool) error {
	if !force && fileExists(destPath) {
		fmt.Printf("yt-dlp already present at %s\n", destPath)
		return nil
	}

	fmt.Printf("Downloading yt-dlp from %s\n", assets.ytdlpURL)
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp("", "spotifydl-ytdlp-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if err := downloadFile(ctx, assets.ytdlpURL, tmpFile); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	if err := copyFile(tmpPath, destPath); err != nil {
		return err
	}

	if runtime.GOOS != "windows" {
		return os.Chmod(destPath, 0o755)
	}

	return nil
}

func downloadFile(ctx context.Context, url string, dest *os.File) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: %s", url, resp.Status)
	}

	_, err = io.Copy(dest, resp.Body)
	return err
}

func extractZip(src, dest string) error {
	reader, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		if err := extractZipFile(file, dest); err != nil {
			return err
		}
	}

	return nil
}

func extractZipFile(file *zip.File, dest string) error {
	target := filepath.Join(dest, file.Name)
	if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dest)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid zip path: %s", file.Name)
	}

	if file.FileInfo().IsDir() {
		return os.MkdirAll(target, 0o755)
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}

	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	return err
}

func extractTarXZ(src, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	xzReader, err := xz.NewReader(file)
	if err != nil {
		return err
	}

	tarReader := tar.NewReader(xzReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, header.Name)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid tar path: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, fs.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}

			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fs.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(out, tarReader); err != nil {
				out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		}
	}
}

func findFFmpegBundleRoot(extractDir string) (string, error) {
	var bundleRoot string

	err := filepath.WalkDir(extractDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() || entry.Name() != "bin" {
			return nil
		}

		if fileExists(filepath.Join(path, "ffmpeg")) || fileExists(filepath.Join(path, "ffmpeg.exe")) {
			bundleRoot = filepath.Dir(path)
			return fs.SkipAll
		}

		return nil
	})
	if err != nil {
		return "", err
	}
	if bundleRoot == "" {
		return "", fmt.Errorf("could not find ffmpeg bin directory in downloaded archive")
	}

	return bundleRoot, nil
}

func copyDir(src, dest string) error {
	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)

		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		return copyFile(path, target)
	})
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fileMode(src))
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func fileMode(src string) fs.FileMode {
	info, err := os.Stat(src)
	if err != nil {
		return 0o644
	}
	return info.Mode().Perm()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
