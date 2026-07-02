package youtube

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/0dev1337/SpotifyDL/pkg/spotify"
	"github.com/bogem/id3v2"
	"golang.org/x/image/webp"
)

func embedMetadata(path string, track spotify.TrackData) error {
	tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err != nil {
		return fmt.Errorf("open mp3 tags: %w", err)
	}
	defer tag.Close()

	tag.SetTitle(track.Name)
	tag.SetArtist(strings.Join(track.ArtistNames(), "; "))
	tag.SetAlbum(track.AlbumOfTrack.Name)

	if albumArtists := track.AlbumArtistNames(); len(albumArtists) > 0 {
		tag.AddTextFrame(tag.CommonID("Band/Orchestra/Accompaniment"), tag.DefaultEncoding(), strings.Join(albumArtists, "; "))
	}

	if track.TrackNumber > 0 {
		tag.AddTextFrame(tag.CommonID("Track number/Position in set"), tag.DefaultEncoding(), fmt.Sprintf("%d", track.TrackNumber))
	}

	if track.DiscNumber > 0 {
		tag.AddTextFrame(tag.CommonID("Part of a set"), tag.DefaultEncoding(), fmt.Sprintf("%d", track.DiscNumber))
	}

	tag.AddCommentFrame(id3v2.CommentFrame{
		Encoding:    tag.DefaultEncoding(),
		Language:    "eng",
		Description: "Spotify",
		Text:        track.URI,
	})

	coverURL := track.LargestCoverURL()
	if coverURL != "" {
		imageData, mimeType, err := fetchCoverArt(coverURL)
		if err != nil {
			return fmt.Errorf("fetch cover art: %w", err)
		}

		imageData, mimeType, err = normalizeCoverImage(imageData, mimeType)
		if err != nil {
			return fmt.Errorf("process cover art: %w", err)
		}

		tag.DeleteFrames(tag.CommonID("Attached picture"))
		tag.AddAttachedPicture(id3v2.PictureFrame{
			Encoding:    id3v2.EncodingUTF8,
			MimeType:    mimeType,
			PictureType: id3v2.PTFrontCover,
			Description: "Album cover",
			Picture:     imageData,
		})
	}

	if err := tag.Save(); err != nil {
		return fmt.Errorf("save mp3 tags: %w", err)
	}

	return nil
}

func fetchCoverArt(url string) ([]byte, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/*,*/*;q=0.8")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("cover art: %s", resp.Status)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, "", err
	}

	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = mimeTypeFromURL(url)
	}
	mimeType = strings.TrimSpace(strings.Split(mimeType, ";")[0])
	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	return data, mimeType, nil
}

func normalizeCoverImage(data []byte, mimeType string) ([]byte, string, error) {
	mimeType = strings.ToLower(strings.TrimSpace(strings.Split(mimeType, ";")[0]))
	switch mimeType {
	case "image/jpeg", "image/jpg":
		return data, "image/jpeg", nil
	case "image/png":
		return data, "image/png", nil
	}

	var (
		img image.Image
		err error
	)
	switch mimeType {
	case "image/webp":
		img, err = webp.Decode(bytes.NewReader(data))
	default:
		img, _, err = image.Decode(bytes.NewReader(data))
	}
	if err != nil {
		return nil, "", err
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		return nil, "", err
	}

	return buf.Bytes(), "image/jpeg", nil
}

func mimeTypeFromURL(url string) string {
	switch strings.ToLower(filepath.Ext(url)) {
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}

func outputPath(track spotify.TrackData) string {
	return filepath.Join(defaultOutputDir, sanitizeFilename(baseFilename(track))+".mp3")
}
