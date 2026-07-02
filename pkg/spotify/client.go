package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"github.com/google/uuid"
)

type AccessTokenResponse struct {
	ClientID                         string `json:"clientId"`
	AccessToken                      string `json:"accessToken"`
	AccessTokenExpirationTimestampMs int64  `json:"accessTokenExpirationTimestampMs"`
	IsAnonymous                      bool   `json:"isAnonymous"`
	Notes                            string `json:"_notes,omitempty"`
}

type ClientTokenResponse struct {
	GrantedTokenResponse GrantedTokenResponse `json:"granted_token"`
}
type GrantedTokenResponse struct {
	Token string `json:"token"`
}
type Client struct {
	httpClient          tls_client.HttpClient
	AccessTokenResponse AccessTokenResponse
	ClientTokenResponse ClientTokenResponse
}

var DefaultPlaylist = "https://open.spotify.com/playlist/46zFIvDlqCi1iNdZVTtGAp" // random hyperpop playlist I found LOL

func NewClient() (*Client, error) {
	jar := tls_client.NewCookieJar()
	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithClientProfile(profiles.Firefox_148),
		tls_client.WithNotFollowRedirects(),
		tls_client.WithCookieJar(jar),
	}

	httpClient, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		return nil, err
	}

	return &Client{httpClient: httpClient}, nil
}

func (c *Client) Setup() error {
	request, _ := http.NewRequest(http.MethodGet, DefaultPlaylist, nil)
	request.Header = DefaultHeaders.Clone()

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("warmup playlist request: %w", err)
	}
	defer response.Body.Close()

	accessTokenResponse, err := c.GetAccessToken()
	if err != nil {
		return fmt.Errorf("get access token: %w", err)
	}

	c.AccessTokenResponse = accessTokenResponse

	clientTokenResponse, err := c.GetClientToken()
	if err != nil {
		return fmt.Errorf("get client token: %w", err)
	}
	c.ClientTokenResponse = clientTokenResponse

	fmt.Println("Access Token: ", c.AccessTokenResponse.AccessToken)
	fmt.Println("Client Token: ", c.ClientTokenResponse.GrantedTokenResponse.Token)
	return nil
}

func (c *Client) GetAccessToken() (AccessTokenResponse, error) {
	result, err := BuildTotp(context.Background(), 0, "")
	if err != nil {
		return AccessTokenResponse{}, err
	}

	request, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("https://open.spotify.com/api/token?reason=init&productType=web-player&totp=%s&totpServer=%s&totpVer=%d", result.Totp, result.TotpServer, result.Version), nil)
	request.Header = DefaultHeaders.Clone()

	response, err := c.httpClient.Do(request)
	if err != nil {
		return AccessTokenResponse{}, err
	}
	defer response.Body.Close()

	var accessTokenResponse AccessTokenResponse
	if err := json.NewDecoder(response.Body).Decode(&accessTokenResponse); err != nil {
		return AccessTokenResponse{}, err
	}

	return accessTokenResponse, nil
}

func (c *Client) GetClientToken() (ClientTokenResponse, error) {

	payload := fmt.Sprintf(`{
  "client_data": {
    "client_version": "1.2.94.189.g1bffdceb",
    "client_id": "%s",
    "js_sdk_data": {
      "device_brand": "unknown",
      "device_model": "unknown",
      "os": "windows",
      "os_version": "NT 10.0",
      "device_id": "%s",
      "device_type": "computer"
    }
  }
}`, c.AccessTokenResponse.ClientID, uuid.New().String())

	request, _ := http.NewRequest(http.MethodPost, "https://clienttoken.spotify.com/v1/clienttoken", strings.NewReader(payload))
	request.Header = DefaultHeaders.Clone()
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return ClientTokenResponse{}, err
	}
	defer response.Body.Close()

	var clientTokenResponse ClientTokenResponse
	if err := json.NewDecoder(response.Body).Decode(&clientTokenResponse); err != nil {
		return ClientTokenResponse{}, err
	}

	return clientTokenResponse, nil
}
