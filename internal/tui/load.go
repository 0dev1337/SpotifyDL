package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/0dev1337/SpotifyDL/internal/tools"
	"github.com/0dev1337/SpotifyDL/pkg/spotify"
	tea "github.com/charmbracelet/bubbletea"
)

type loadProgressMsg struct {
	detail string
	clear  bool
}

type loadStartedMsg struct {
	ch chan tea.Msg
}

func loadPlaylist(playlistID string) tea.Cmd {
	ch := make(chan tea.Msg, 8)
	go runLoadPlaylist(ch, playlistID)
	return func() tea.Msg {
		return loadStartedMsg{ch: ch}
	}
}

func runLoadPlaylist(ch chan tea.Msg, playlistID string) {
	onProgress := func(msg string) {
		ch <- loadProgressMsg{detail: msg}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
	defer cancel()

	if _, err := tools.ResolveOrInstallWithProgress(ctx, "", "", onProgress); err != nil {
		ch <- loadDoneMsg{err: fmt.Errorf("dependencies: %w", err)}
		return
	}
	ch <- loadProgressMsg{clear: true}

	client, err := spotify.NewClient()
	if err != nil {
		ch <- loadDoneMsg{err: fmt.Errorf("create client: %w", err)}
		return
	}

	if err := client.Setup(); err != nil {
		ch <- loadDoneMsg{err: fmt.Errorf("setup client: %w", err)}
		return
	}

	playlist, err := client.GetPlaylist(playlistID)
	if err != nil {
		ch <- loadDoneMsg{err: fmt.Errorf("fetch playlist: %w", err)}
		return
	}

	ch <- loadDoneMsg{playlist: playlist}
}

func waitForLoad(ch chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}
