package tui

import (
	"github.com/0dev1337/SpotifyDL/pkg/spotify"
	"github.com/0dev1337/SpotifyDL/pkg/youtube"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/zenthangplus/goccm"
)

type trackStartedMsg struct {
	track spotify.TrackData
}

type trackCompletedMsg struct {
	track spotify.TrackData
	err   error
}

type downloadFinishedMsg struct{}

func runConcurrentDownloads(ch chan tea.Msg, tracks []spotify.TrackData, workers int) {
	ccm := goccm.New(workers)

	for _, track := range tracks {
		ccm.Wait()
		go func(t spotify.TrackData) {
			defer ccm.Done()
			ch <- trackStartedMsg{track: t}
			ch <- trackCompletedMsg{track: t, err: youtube.DownloadMusic(t)}
		}(track)
	}

	ccm.WaitAllDone()
	ch <- downloadFinishedMsg{}
}

func waitForDownload(ch chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}
