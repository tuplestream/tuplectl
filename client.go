package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"runtime"
)

var client = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

func userAgent() string {
	return fmt.Sprintf("tuplectl-%s-%s-%s", version(), runtime.GOOS, runtime.GOARCH)
}

func baseURL() string {
	userSpecified := os.Getenv("TUPLECTL_CONTROL_API_BASE_URL")
	if userSpecified != "" {
		url, err := url.ParseRequestURI(userSpecified)
		handleError(err)
		if url.Scheme == "http" {
			warn("using insecure base url for API calls. Consider using an 'https' endpoint.")
		}
		return userSpecified
	}
	return "https://api.tuplestream.com"
}

func baseRequest(method string, path string) *http.Request {
	if accessToken == "" {
		doAuth()
	}
	target := baseURL() + path
	debug("Calling " + target)
	req, err := http.NewRequest(method, target, nil)
	handleError(err)
	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("User-Agent", userAgent())
	return req
}

func getResource(resource string) (*http.Response, error) {
	req := baseRequest("GET", resource)
	return client.Do(req)
}
