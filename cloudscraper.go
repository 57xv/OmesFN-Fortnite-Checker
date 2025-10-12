package main

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

type roundTripper struct {
	transport http.RoundTripper
	userAgent string
}

func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", rt.userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Connection", "keep-alive")
	return rt.transport.RoundTrip(req)
}

func NewCloudScraper(proxy string) *http.Client {
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36"

	var transport http.RoundTripper = http.DefaultTransport

	if proxy != "" {
		if !strings.Contains(proxy, "://") {
			proxy = "http://" + proxy
		}
		proxyURL, err := url.Parse(proxy)
		if err == nil {
			transport = &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
		}
	}

	// Create a cookie jar
	jar, _ := cookiejar.New(nil)

	return &http.Client{
		Transport: &roundTripper{
			transport: transport,
			userAgent: userAgent,
		},
		Jar: jar,
	}
}

func ApplyCloudScraperHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
}
