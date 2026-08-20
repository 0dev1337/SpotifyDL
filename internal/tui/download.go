package tui

import (
	"sync"
	"time"

	"github.com/0dev1337/SpotifyDL/internal/tools"
	"github.com/0dev1337/SpotifyDL/pkg/spotify"
	"github.com/0dev1337/SpotifyDL/pkg/youtube"
	tea "github.com/charmbracelet/bubbletea"
)

type trackStartedMsg struct {
	track spotify.TrackData
}

type trackCompletedMsg struct {
	track spotify.TrackData
	err   error
}

type downloadFinishedMsg struct{}

type downloadTickMsg struct{}

func downloadTick() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return downloadTickMsg{}
	})
}

func runConcurrentDownloads(ch chan tea.Msg, tracks []spotify.TrackData, workers int, paths tools.Paths) {
	if len(tracks) == 0 {
		ch <- downloadFinishedMsg{}
		return
	}

	workers = clampWorkers(workers)
	if workers > len(tracks) {
		workers = len(tracks)
	}

	jobs := make(chan spotify.TrackData, len(tracks))
	for _, track := range tracks {
		jobs <- track
	}
	close(jobs)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for track := range jobs {
				ch <- trackStartedMsg{track: track}
				ch <- trackCompletedMsg{track: track, err: youtube.DownloadMusicWithPaths(track, paths)}
			}
		}()
	}

	wg.Wait()
	ch <- downloadFinishedMsg{}
}

func waitForDownload(ch chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

func clampWorkers(workers int) int {
	if workers < minWorkers {
		return minWorkers
	}
	if workers > maxWorkers {
		return maxWorkers
	}
	return workers
}
