package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

var ytdlpSetupMu sync.Mutex

func withYTDLPSetupLock(ctx context.Context, root string, fn func() error) error {
	ytdlpSetupMu.Lock()
	defer ytdlpSetupMu.Unlock()

	lockPath, err := ytdlpLockPath(root)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("open yt-dlp lock: %w", err)
	}
	defer file.Close()

	for {
		if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err == nil {
			break
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for other yt-dlp processes to finish")
		case <-time.After(500 * time.Millisecond):
		}
	}
	defer func() { _ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN) }()

	return fn()
}

func ytdlpLockPath(root string) (string, error) {
	dir, err := ToolsDir(root)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ".yt-dlp.lock"), nil
}

func defaultInstallRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	if moduleRoot := findModuleRoot(wd); moduleRoot != "" {
		return moduleRoot
	}
	return wd
}
