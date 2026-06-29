package spotify

import (
	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

type AccessTokenResponse struct {
	ClientID                         string `json:"clientId"`
	AccessToken                      string `json:"accessToken"`
	AccessTokenExpirationTimestampMs int64  `json:"accessTokenExpirationTimestampMs"`
	IsAnonymous                      bool   `json:"isAnonymous"`
	Notes                            string `json:"_notes,omitempty"`
}
type Spotify struct {
	Client              tls_client.HttpClient
	AccessTokenResponse AccessTokenResponse
}

var (
	DefaultPlaylist = "https://open.spotify.com/playlist/46zFIvDlqCi1iNdZVTtGAp" // random hyperpop playlist I found LOL
)

func SetupClient() error {
	jar := tls_client.NewCookieJar()
	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithClientProfile(profiles.Firefox_148),
		tls_client.WithNotFollowRedirects(),
		tls_client.WithCookieJar(jar), // create cookieJar instance and pass it as argument
	}
	var Spotify Spotify

	Spotify.Client, _ = tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)

	request, _ := http.NewRequest(http.MethodGet, DefaultPlaylist, nil)

	request.Header = DefaultHeaders.Clone()
	response, err := Spotify.Client.Do(request)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	return nil
}

// func GetAccessToken() (AccessTokenResponse, error) {
// 	request , _ := http.NewRequest(http.MethodGet, "https://open.spotify.com/api/token?reason=init&productType=web-player&totp=465536&totpServer=465536&totpVer=61", nil)
// }
