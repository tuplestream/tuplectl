package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"runtime"
)

var client = &http.Client{}

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
	return "https://api.tuplestream.net"
}

func baseRequest(method string, path string) *http.Request {
	req, err := http.NewRequest(method, baseURL()+path, nil)
	handleError(err)
	req.Header.Add("Authorization", "Bearer "+jwt)
	req.Header.Add("User-Agent", userAgent())
	return req
}

func execute(req *http.Request) string {
	resp, err := client.Do(req)
	handleError(err)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	handleError(err)
	return string(body)
}

func getResource(resource string) string {
	req := baseRequest("GET", "/"+resource)
	return execute(req)
}
