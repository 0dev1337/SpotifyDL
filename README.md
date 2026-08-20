# SpotifyDL

Download any Spotify playlist as MP3 files from your terminal. Paste a link, wait for the download to finish, and find tagged tracks with album art in the `downloads/` folder.



> **Demo:** <img width="1481" height="1105" alt="WindowsTerminal_APSsHINotu" src="https://github.com/user-attachments/assets/63bd5f61-a33b-42d5-8bd9-44983bb7c70e" />


## Quick start

**Requirements:** [Go 1.25+](https://go.dev/dl/) and a modern terminal (Windows Terminal, iTerm2, etc.)

```bash
git clone https://github.com/0dev1337/SpotifyDL.git
cd SpotifyDL
go run ./cmd/main.go
```

On first run, SpotifyDL downloads `ffmpeg` and `yt-dlp` automatically if they are not already installed.

## User guide

### 1. Open the app

Run `go run ./cmd/main.go` (or `./spotifydl` if you built the binary). You will see the main menu.

### 2. Download a playlist

1. Select **Download Playlist**
2. Paste a Spotify playlist URL or ID, for example:
   ```
   https://open.spotify.com/playlist/37i9dQZF1DXcBWIGoYBM5M
   ```
3. Press **Enter**
4. Wait while the playlist loads and tracks download
5. When finished, press **Enter** to return to the menu

### 3. Adjust download speed (optional)

1. From the main menu, open **Settings**
2. Use **↑ / ↓** or **+ / -** to change concurrent workers (default: 4, range: 1–50). Use **PgUp / PgDn** for ±10.
3. Press **Enter** to save and go back

More workers download faster but use more bandwidth and CPU.

### 4. Find your files

MP3s are saved to:

```
downloads/Artist Name - Track Title.mp3
```

Each file includes Spotify metadata: title, artist, album, and cover art.

## Controls

| Key | Action |
|-----|--------|
| `↑` `↓` | Navigate menu / change settings |
| `Enter` | Select or confirm |
| `Esc` | Go back |
| `q` | Quit |

## Build a standalone binary

```bash
go build -o spotifydl ./cmd/main.go
./spotifydl
```

## Troubleshooting

| Issue | What to try |
|-------|-------------|
| Downloads fail with YouTube errors | Install [Deno](https://deno.com/) or [Node.js](https://nodejs.org/) so yt-dlp can solve YouTube challenges |
| No cover art on old files | Re-download the track — art is embedded during download |
| Tools not found | Delete the `tools/` folder and run again to re-download dependencies |

## Disclaimer

For personal use only. Downloading copyrighted music may violate Spotify's Terms of Service and local laws. Use responsibly.

## License

[MIT](LICENSE)
