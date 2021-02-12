package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

// request an ingestion token from the API.
func getIngestToken() string {
	resp, err := createResource("/tokens", "")
	handleError(err)
	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	handleError(err)
	var tokenResponse map[string]string
	err = json.Unmarshal(bytes, &tokenResponse)
	handleError(err)
	if tokenResponse["token"] == "" {
		panic("Got empty token response from API")
	}
	return tokenResponse["token"]
}

func deploy() {
	app := "kubectl"
	arg0 := "kustomize"
	arg1 := "github.com/tuplestream/collector/kustomize"

	cmd := exec.Command(app, arg0, arg1)
	stdout, err := cmd.Output()

	if err != nil {
		fmt.Println("Error invoking kubectl!\n" + err.Error())
		os.Exit(1)
	}

	token := getIngestToken()
	// JWTs are base64URL encoded, we need to encode as base64 to
	// store as a kubernetes secret
	base64Encoded := base64.StdEncoding.EncodeToString([]byte(token))
	// 'TODO' in base64, which got encoded via kustomize
	placeholder := "VE9ETw=="

	withToken := strings.Replace(string(stdout), placeholder, base64Encoded, 1)

	fmt.Print(withToken)
}
