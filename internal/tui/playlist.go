package tui

import "strings"

func parsePlaylistID(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	if strings.HasPrefix(input, "spotify:playlist:") {
		return strings.TrimPrefix(input, "spotify:playlist:")
	}

	if idx := strings.Index(input, "playlist/"); idx >= 0 {
		id := input[idx+len("playlist/"):]
		if end := strings.IndexAny(id, "?#"); end >= 0 {
			id = id[:end]
		}
		return id
	}

	return input
}
