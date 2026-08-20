package tools

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// WarmupYTDLP pre-fetches JS challenge components once so concurrent downloads
// don't all block trying to install them at the same time.
func WarmupYTDLP(ctx context.Context, paths Paths, onProgress ProgressFunc) error {
	if onProgress != nil {
		onProgress("Preparing yt-dlp...")
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	return withYTDLPSetupLock(ctx, defaultInstallRoot(), func() error {
		args := []string{"--remote-components", "ejs:github", "--version"}
		args = appendJSRuntimeArgs(args)

		cmd := exec.CommandContext(ctx, paths.YTDLP, args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			detail := string(out)
			if len(detail) > 200 {
				detail = detail[:200] + "..."
			}
			return fmt.Errorf("yt-dlp warmup: %w (%s)", err, detail)
		}
		return nil
	})
}

func appendJSRuntimeArgs(args []string) []string {
	for _, runtime := range []string{"deno", "node"} {
		if path, err := exec.LookPath(runtime); err == nil {
			return append(args, "--js-runtimes", runtime+":"+path)
		}
	}
	return args
}
