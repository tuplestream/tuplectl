package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
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

func baseRequest(method string, path string, data string) *http.Request {
	if accessToken == "" {
		doAuth()
	}
	target := baseURL() + path
	debug("Calling " + target)
	var req *http.Request
	var err error
	if data == "" {
		req, err = http.NewRequest(method, target, nil)
	} else {
		req, err = http.NewRequest(method, target, bytes.NewBuffer([]byte(data)))
	}

	handleError(err)
	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("User-Agent", userAgent())
	return req
}

func getResource(resource string) (*http.Response, error) {
	req := baseRequest("GET", resource, "")
	return client.Do(req)
}

func getResourceString(resource string) string {
	res, err := getResource(resource)
	handleError(err)
	defer res.Body.Close()
	bytes, err := ioutil.ReadAll(res.Body)
	handleError(err)
	return string(bytes)
}

func deleteResource(resource string) (*http.Response, error) {
	req := baseRequest("DELETE", resource, "")
	return client.Do(req)
}

func createResource(resource string, data string) (*http.Response, error) {
	req := baseRequest("POST", resource, data)
	return client.Do(req)
}
