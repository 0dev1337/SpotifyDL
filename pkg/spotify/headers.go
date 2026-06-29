package spotify

import http "github.com/bogdanfinn/fhttp"

var (
	DefaultHeaders = http.Header{
		"Host":                      {"open.spotify.com"},
		"Connection":                {"keep-alive"},
		"Upgrade-Insecure-Requests": {"1"},
		"User-Agent":                {"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:148.0) Gecko/20100101 Firefox/148.0"},
		"Accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"},
		"Accept-Encoding":           {"gzip, deflate, br, zstd"},
		"Accept-Language":           {"en-US,en;q=0.9"},
		"Priority":                  {"u=0, i"},
		"Sec-Fetch-Dest":            {"document"},
		"Sec-Fetch-Mode":            {"navigate"},
		"Sec-Fetch-Site":            {"none"},
		"Sec-Fetch-User":            {"?1"},

		http.HeaderOrderKey: {
			"Host",
			"Connection",
			"Upgrade-Insecure-Requests",
			"User-Agent",
			"Accept",
			"Accept-Encoding",
			"Accept-Language",
			"Priority",
			"Sec-Fetch-Dest",
			"Sec-Fetch-Mode",
			"Sec-Fetch-Site",
			"Sec-Fetch-User",
		},
	}
)
