package spotify

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const spotifyOrigin = "https://open.spotify.com"

var (
	bundleURLPattern  = regexp.MustCompile(`https?://[^"'\s]+/cdn/build/web-player/web-player\.[0-9a-f]{8,}\.js`)
	secretPairPattern = regexp.MustCompile(`\{secret:'((?:\\.|[^'\\])*)',version:(\d+)\}`)
)

type SecretEntry struct {
	Version int    `json:"version"`
	Cipher  string `json:"cipher"`
}

type TotpResult struct {
	Version    int    `json:"version"`
	Totp       string `json:"totp"`
	TotpServer string `json:"totpServer"`
	ClientTime int64  `json:"clientTime"`
	ServerTime *int64 `json:"serverTime,omitempty"`
	Cipher     string `json:"-"`
}

type serverTimeResponse struct {
	ServerTime int64 `json:"serverTime"`
}

func FetchBundleURL(ctx context.Context) (string, error) {
	body, err := httpGet(ctx, spotifyOrigin+"/", 30*time.Second)
	if err != nil {
		return "", err
	}

	match := bundleURLPattern.FindString(string(body))
	if match == "" {
		return "", fmt.Errorf("could not find web-player bundle URL in open.spotify.com HTML")
	}
	return match, nil
}

func FetchSecrets(ctx context.Context, bundleURL string) ([]SecretEntry, error) {
	if bundleURL == "" {
		var err error
		bundleURL, err = FetchBundleURL(ctx)
		if err != nil {
			return nil, err
		}
	}

	body, err := httpGet(ctx, bundleURL, 30*time.Second)
	if err != nil {
		return nil, err
	}

	entries := make(map[int]SecretEntry)
	for _, match := range secretPairPattern.FindAllStringSubmatch(string(body), -1) {
		cipher := unescapeJSString(match[1])
		version, err := strconv.Atoi(match[2])
		if err != nil {
			return nil, fmt.Errorf("parse secret version: %w", err)
		}
		entries[version] = SecretEntry{Version: version, Cipher: cipher}
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no TOTP secrets found in web-player bundle")
	}

	versions := make([]int, 0, len(entries))
	for version := range entries {
		versions = append(versions, version)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(versions)))

	secrets := make([]SecretEntry, len(versions))
	for i, version := range versions {
		secrets[i] = entries[version]
	}
	return secrets, nil
}

func PickActiveSecret(secrets []SecretEntry, version int) (SecretEntry, error) {
	if version != 0 {
		for _, entry := range secrets {
			if entry.Version == version {
				return entry, nil
			}
		}
		known := make([]string, len(secrets))
		for i, entry := range secrets {
			known[i] = strconv.Itoa(entry.Version)
		}
		return SecretEntry{}, fmt.Errorf("requested totpVer %d not found (available: %s)", version, strings.Join(known, ", "))
	}
	if len(secrets) == 0 {
		return SecretEntry{}, fmt.Errorf("no secrets available")
	}
	return secrets[0], nil
}

func DecodeSecret(cipher string) []byte {
	transformed := make([]byte, len(cipher))
	for i, char := range cipher {
		transformed[i] = byte(char ^ rune((i%33)+9))
	}

	var b strings.Builder
	for _, value := range transformed {
		b.WriteString(strconv.Itoa(int(value)))
	}
	return []byte(b.String())
}

func GenerateTotp(secret []byte, timestamp int64) string {
	counter := uint64(timestamp / 30)

	mac := hmac.New(sha1.New, secret)
	var counterBytes [8]byte
	binary.BigEndian.PutUint64(counterBytes[:], counter)
	_, _ = mac.Write(counterBytes[:])
	digest := mac.Sum(nil)

	offset := digest[len(digest)-1] & 0x0F
	code := binary.BigEndian.Uint32(digest[offset:offset+4]) & 0x7FFFFFFF
	return fmt.Sprintf("%06d", code%1_000_000)
}

func FetchServerTime(ctx context.Context) (*int64, error) {
	body, headers, err := httpGetWithHeaders(ctx, spotifyOrigin+"/api/server-time", 5*time.Second)
	if err == nil {
		var payload serverTimeResponse
		if json.Unmarshal(body, &payload) == nil && payload.ServerTime > 0 {
			return &payload.ServerTime, nil
		}
	}

	_, headers, err = httpGetWithHeaders(ctx, spotifyOrigin+"/", 5*time.Second)
	if err != nil {
		return nil, nil
	}

	dateHeader := headers.Get("Date")
	if dateHeader == "" {
		return nil, nil
	}

	parsed, err := http.ParseTime(dateHeader)
	if err != nil {
		return nil, nil
	}

	unix := parsed.Unix()
	return &unix, nil
}

func BuildTotp(ctx context.Context, version int, bundleURL string) (TotpResult, error) {
	secrets, err := FetchSecrets(ctx, bundleURL)
	if err != nil {
		return TotpResult{}, err
	}

	active, err := PickActiveSecret(secrets, version)
	if err != nil {
		return TotpResult{}, err
	}

	secretBytes := DecodeSecret(active.Cipher)
	clientTime := time.Now().Unix()
	serverTime, _ := FetchServerTime(ctx)

	totp := GenerateTotp(secretBytes, clientTime)
	totpServer := totp
	if serverTime != nil {
		totpServer = GenerateTotp(secretBytes, *serverTime)
	}

	return TotpResult{
		Version:    active.Version,
		Totp:       totp,
		TotpServer: totpServer,
		ClientTime: clientTime,
		ServerTime: serverTime,
		Cipher:     active.Cipher,
	}, nil
}

func httpGet(ctx context.Context, url string, timeout time.Duration) ([]byte, error) {
	body, _, err := httpGetWithHeaders(ctx, url, timeout)
	return body, err
}

func httpGetWithHeaders(ctx context.Context, url string, timeout time.Duration) ([]byte, http.Header, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("User-Agent", defaultUserAgent())
	req.Header.Set("Accept", "*/*")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, resp.Header, fmt.Errorf("GET %s: %s", url, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.Header, err
	}
	return body, resp.Header, nil
}

func defaultUserAgent() string {
	if values := DefaultHeaders["User-Agent"]; len(values) > 0 {
		return values[0]
	}
	return "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:148.0) Gecko/20100101 Firefox/148.0"
}

func unescapeJSString(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			continue
		}

		switch s[i+1] {
		case '\\', '\'', '"':
			b.WriteByte(s[i+1])
			i++
		case 'n':
			b.WriteByte('\n')
			i++
		case 'r':
			b.WriteByte('\r')
			i++
		case 't':
			b.WriteByte('\t')
			i++
		case 'u':
			if i+5 < len(s) {
				if r, err := strconv.ParseUint(s[i+2:i+6], 16, 32); err == nil {
					b.WriteRune(rune(r))
					i += 5
					continue
				}
			}
			b.WriteByte(s[i])
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
