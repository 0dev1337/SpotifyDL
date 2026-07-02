package tui

import (
	"fmt"
	"strings"

	"github.com/0dev1337/SpotifyDL/pkg/spotify"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type page int

const (
	pageMenu page = iota
	pageSettings
	pagePlaylistInput
	pageLoading
	pageDownload
	pageDone
)

const (
	defaultWorkers = 4
	minWorkers     = 1
	maxWorkers     = 16
)

type model struct {
	page          page
	width         int
	height        int
	menuCursor    int
	menuItems     []string
	textInput     textinput.Model
	spinner       spinner.Model
	loadingText   string
	loadingDetail string
	loadCh        chan tea.Msg
	playlistID    string
	playlistName  string
	tracks        []spotify.TrackData
	workers       int
	succeeded     int
	failed        int
	activeTracks  map[string]string
	latestTrack   string
	latestArtist  string
	downloadCh    chan tea.Msg
	statusMsg     string
}

type loadDoneMsg struct {
	playlist *spotify.PlaylistResponse
	err      error
}

func Run() error {
	p := tea.NewProgram(newModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func newModel() model {
	ti := textinput.New()
	ti.Placeholder = "https://open.spotify.com/playlist/..."
	ti.CharLimit = 256
	ti.Width = 50
	ti.PromptStyle = lipgloss.NewStyle().Foreground(colorGreen)
	ti.TextStyle = lipgloss.NewStyle().Foreground(colorWhite)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorGreen)

	return model{
		page:      pageMenu,
		menuItems: []string{"Download Playlist", "Settings", "Quit"},
		textInput: ti,
		spinner:   s,
		workers:   defaultWorkers,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch m.page {
		case pageDone:
			if msg.String() == "enter" || msg.String() == "q" || msg.String() == "esc" {
				return m.resetToMenu(), nil
			}
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			return m, nil
		}

		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}

		switch m.page {
		case pageMenu:
			return m.updateMenu(msg)
		case pageSettings:
			return m.updateSettings(msg)
		case pagePlaylistInput:
			return m.updatePlaylistInput(msg)
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case loadStartedMsg:
		m.loadCh = msg.ch
		m.loadingText = "Preparing..."
		m.loadingDetail = ""
		return m, waitForLoad(msg.ch)

	case loadProgressMsg:
		if msg.clear {
			m.loadingDetail = ""
			m.loadingText = "Fetching playlist..."
		} else {
			m.loadingDetail = msg.detail
		}
		return m, waitForLoad(m.loadCh)

	case loadDoneMsg:
		m.loadCh = nil
		m.loadingDetail = ""
		if msg.err != nil {
			m.page = pagePlaylistInput
			m.statusMsg = msg.err.Error()
			m.textInput.Focus()
			return m, textinput.Blink
		}
		return m.startDownload(msg.playlist)

	case trackStartedMsg:
		label := fmt.Sprintf("%s — %s", msg.track.Name, strings.Join(msg.track.ArtistNames(), ", "))
		if m.activeTracks == nil {
			m.activeTracks = make(map[string]string)
		}
		m.activeTracks[msg.track.URI] = label
		return m, waitForDownload(m.downloadCh)

	case trackCompletedMsg:
		delete(m.activeTracks, msg.track.URI)
		m.latestTrack = msg.track.Name
		m.latestArtist = strings.Join(msg.track.ArtistNames(), ", ")

		if msg.err != nil {
			m.failed++
		} else {
			m.succeeded++
		}
		return m, waitForDownload(m.downloadCh)

	case downloadFinishedMsg:
		m.page = pageDone
		m.activeTracks = nil
		return m, nil
	}

	return m, nil
}

func (m model) startDownload(playlist *spotify.PlaylistResponse) (model, tea.Cmd) {
	m.tracks = playlist.Tracks()
	m.playlistName = playlist.Data.PlaylistV2.Name
	m.succeeded = 0
	m.failed = 0
	m.activeTracks = nil
	m.latestTrack = ""
	m.latestArtist = ""

	if len(m.tracks) == 0 {
		m.page = pageDone
		m.statusMsg = "Playlist has no tracks."
		return m, nil
	}

	m.page = pageDownload
	m.downloadCh = make(chan tea.Msg, m.workers*4)
	go runConcurrentDownloads(m.downloadCh, m.tracks, m.workers)
	return m, waitForDownload(m.downloadCh)
}

func (m model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.menuCursor > 0 {
			m.menuCursor--
		}
	case "down", "j":
		if m.menuCursor < len(m.menuItems)-1 {
			m.menuCursor++
		}
	case "enter":
		switch m.menuItems[m.menuCursor] {
		case "Download Playlist":
			m.page = pagePlaylistInput
			m.statusMsg = ""
			m.textInput.SetValue("")
			m.textInput.Focus()
			return m, textinput.Blink
		case "Settings":
			m.page = pageSettings
			return m, nil
		case "Quit":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) updateSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.page = pageMenu
		return m, nil
	case "up", "k", "+":
		if m.workers < maxWorkers {
			m.workers++
		}
	case "down", "j", "-":
		if m.workers > minWorkers {
			m.workers--
		}
	}
	return m, nil
}

func (m model) updatePlaylistInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.page = pageMenu
		m.textInput.Blur()
		return m, nil
	case "enter":
		id := parsePlaylistID(m.textInput.Value())
		if id == "" {
			m.statusMsg = "Enter a playlist URL or ID."
			return m, nil
		}
		m.playlistID = id
		m.page = pageLoading
		m.loadingText = "Preparing..."
		m.loadingDetail = ""
		m.textInput.Blur()
		return m, tea.Batch(m.spinner.Tick, loadPlaylist(id))
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) resetToMenu() model {
	m.page = pageMenu
	m.menuCursor = 0
	m.statusMsg = ""
	m.tracks = nil
	m.succeeded = 0
	m.failed = 0
	m.activeTracks = nil
	m.downloadCh = nil
	m.loadCh = nil
	m.loadingDetail = ""
	return m
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var content string
	switch m.page {
	case pageMenu:
		content = renderMenu(m)
	case pageSettings:
		content = renderSettings(m)
	case pagePlaylistInput:
		content = renderPlaylistInput(m)
	case pageLoading:
		content = renderLoading(m)
	case pageDownload:
		content = renderDownload(m)
	case pageDone:
		content = renderDone(m)
	}

	return lipgloss.Place(
		m.width,
		max(10, m.height),
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

func renderMenu(m model) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("SpotifyDL"))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("Download Spotify playlists as MP3"))
	b.WriteString("\n\n")
	b.WriteString(boxStyle.Render(renderMenuItems(m)))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/↓ navigate  •  enter select  •  q quit"))
	return b.String()
}

func renderSettings(m model) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Settings"))
	b.WriteString("\n\n")
	b.WriteString(boxStyle.Width(40).Render(strings.Join([]string{
		labelStyle.Render("Concurrent downloads"),
		valueStyle.Render(fmt.Sprintf("%d workers", m.workers)),
		subtitleStyle.Render(fmt.Sprintf("Range: %d–%d", minWorkers, maxWorkers)),
	}, "\n")))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/↓ or +/- adjust  •  enter back  •  esc back"))
	return b.String()
}

func renderMenuItems(m model) string {
	var lines []string
	for i, item := range m.menuItems {
		if i == m.menuCursor {
			lines = append(lines, menuSelectedStyle.Render("▸ "+item))
		} else {
			lines = append(lines, menuItemStyle.Render("  "+item))
		}
	}
	return strings.Join(lines, "\n")
}

func renderPlaylistInput(m model) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Playlist URL"))
	b.WriteString("\n\n")
	b.WriteString(labelStyle.Render("Paste a Spotify playlist link or ID"))
	b.WriteString("\n\n")
	b.WriteString(m.textInput.View())
	if m.statusMsg != "" {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(m.statusMsg))
	}
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("enter submit  •  esc back"))
	return b.String()
}

func renderLoading(m model) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Loading"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("%s %s", m.spinner.View(), labelStyle.Render(m.loadingText)))
	if m.loadingDetail != "" {
		b.WriteString("\n")
		b.WriteString(subtitleStyle.Render(m.loadingDetail))
	}
	return b.String()
}

func renderDownload(m model) string {
	total := len(m.tracks)
	completed := m.succeeded + m.failed
	fraction := 0.0
	if total > 0 {
		fraction = float64(completed) / float64(total)
	}

	barWidth := min(50, max(20, m.width-24))
	lines := []string{
		labelStyle.Render("Playlist"),
		valueStyle.Render(m.playlistName),
		"",
		labelStyle.Render("Progress"),
		renderProgressBar(fraction, barWidth),
		progressLine(completed, total, m.workers),
		"",
		labelStyle.Render("In progress"),
	}
	if len(m.activeTracks) == 0 {
		lines = append(lines, subtitleStyle.Render("starting workers..."))
	} else {
		for _, track := range m.activeTracks {
			lines = append(lines, subtitleStyle.Render("• "+track))
		}
	}
	if m.latestTrack != "" {
		lines = append(lines, "", labelStyle.Render("Latest finished"))
		lines = append(lines, valueStyle.Render(m.latestTrack))
		lines = append(lines, subtitleStyle.Render(m.latestArtist))
	}
	lines = append(lines, "",
		successStyle.Render(fmt.Sprintf("✓ %d succeeded", m.succeeded))+"   "+
			errorStyle.Render(fmt.Sprintf("✗ %d failed", m.failed)),
	)

	var b strings.Builder
	b.WriteString(titleStyle.Render("Downloading"))
	b.WriteString("\n\n")
	b.WriteString(boxStyle.Width(min(64, m.width-10)).Render(strings.Join(lines, "\n")))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("please wait...  •  ctrl+c quit"))
	return b.String()
}

func renderDone(m model) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Finished"))
	b.WriteString("\n\n")

	lines := []string{
		valueStyle.Render(m.playlistName),
		"",
		successStyle.Render("Download complete."),
	}
	if m.statusMsg != "" {
		lines = append(lines, "", subtitleStyle.Render(m.statusMsg))
	}

	b.WriteString(boxStyle.Width(min(64, m.width-10)).Render(strings.Join(lines, "\n")))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("enter back to menu  •  q quit"))
	return b.String()
}
