package spotify

import (
	"strings"
)

type PlaylistResponse struct {
	Data PlaylistData `json:"data"`
}

type PlaylistData struct {
	PlaylistV2 PlaylistV2 `json:"playlistV2"`
}

type PlaylistV2 struct {
	Typename    string              `json:"__typename"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	URI         string              `json:"uri"`
	Followers   int                 `json:"followers"`
	Content     PlaylistContentPage `json:"content"`
	SharingInfo SharingInfo         `json:"sharingInfo"`
}

type SharingInfo struct {
	ShareID  string `json:"shareId"`
	ShareURL string `json:"shareUrl"`
}

type PlaylistContentPage struct {
	Typename   string         `json:"__typename"`
	Items      []PlaylistItem `json:"items"`
	PagingInfo PagingInfo     `json:"pagingInfo"`
	TotalCount int            `json:"totalCount"`
}

type PagingInfo struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type PlaylistItem struct {
	UID     string         `json:"uid"`
	AddedAt AddedAt        `json:"addedAt"`
	ItemV2  TrackWrapperV2 `json:"itemV2"`
}

type AddedAt struct {
	ISOString string `json:"isoString"`
}

type TrackWrapperV2 struct {
	Typename string    `json:"__typename"`
	Data     TrackData `json:"data"`
}

type TrackData struct {
	Typename      string        `json:"__typename"`
	Name          string        `json:"name"`
	URI           string        `json:"uri"`
	TrackNumber   int           `json:"trackNumber"`
	DiscNumber    int           `json:"discNumber"`
	Playcount     string        `json:"playcount"`
	MediaType     string        `json:"mediaType"`
	TrackDuration TrackDuration `json:"trackDuration"`
	Artists       ArtistItems   `json:"artists"`
	AlbumOfTrack  AlbumOfTrack  `json:"albumOfTrack"`
	Playability   Playability   `json:"playability"`
	ContentRating ContentRating `json:"contentRating"`
}

type TrackDuration struct {
	TotalMilliseconds int `json:"totalMilliseconds"`
}

type ArtistItems struct {
	Items []ArtistItem `json:"items"`
}

type ArtistItem struct {
	URI     string         `json:"uri"`
	Profile ArtistProfile  `json:"profile"`
}

type ArtistProfile struct {
	Name string `json:"name"`
}

type AlbumOfTrack struct {
	Name     string   `json:"name"`
	URI      string   `json:"uri"`
	Artists  ArtistItems `json:"artists"`
	CoverArt CoverArt `json:"coverArt"`
}

type CoverArt struct {
	Sources []ImageSource `json:"sources"`
}

type ImageSource struct {
	Height int    `json:"height"`
	Width  int    `json:"width"`
	URL    string `json:"url"`
}

type Playability struct {
	Playable bool   `json:"playable"`
	Reason   string `json:"reason"`
}

type ContentRating struct {
	Label string `json:"label"`
}

func (p *PlaylistResponse) Tracks() []TrackData {
	if p == nil {
		return nil
	}

	items := p.Data.PlaylistV2.Content.Items
	tracks := make([]TrackData, 0, len(items))
	for _, item := range items {
		if item.ItemV2.Data.URI == "" {
			continue
		}
		tracks = append(tracks, item.ItemV2.Data)
	}
	return tracks
}

func (t TrackData) TrackID() string {
	const prefix = "spotify:track:"
	if strings.HasPrefix(t.URI, prefix) {
		return strings.TrimPrefix(t.URI, prefix)
	}
	return t.URI
}

func (t TrackData) ArtistNames() []string {
	names := make([]string, 0, len(t.Artists.Items))
	for _, artist := range t.Artists.Items {
		if artist.Profile.Name != "" {
			names = append(names, artist.Profile.Name)
		}
	}
	return names
}

func (t TrackData) AlbumArtistNames() []string {
	names := make([]string, 0, len(t.AlbumOfTrack.Artists.Items))
	for _, artist := range t.AlbumOfTrack.Artists.Items {
		if artist.Profile.Name != "" {
			names = append(names, artist.Profile.Name)
		}
	}
	return names
}

func (t TrackData) LargestCoverURL() string {
	var best ImageSource
	for _, source := range t.AlbumOfTrack.CoverArt.Sources {
		if source.URL == "" {
			continue
		}

		size := sourceSize(source)
		bestSize := sourceSize(best)
		if best.URL == "" || size > bestSize {
			best = source
		}
	}
	return best.URL
}

func sourceSize(source ImageSource) int {
	if source.Width > 0 {
		return source.Width
	}
	return source.Height
}
