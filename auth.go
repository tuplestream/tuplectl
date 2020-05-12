package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/zalando/go-keyring"
)

type CodeResponse struct {
	VerificationUriComplete string `json:"verification_uri_complete"`
	DeviceCode              string `json:"device_code"`
	Interval                int    `json:"interval"`
	Expires                 int    `json:"expires_in"`
}

type JwtError struct {
	Error string `json:"error"`
}

type JwtSuccess struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

func openbrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		fmt.Println("To continue authentication, open this url in a browser: " + url)
	}

	handleError(err)
}

var clientID = getEnvOrDefault("TUPLECTL_AUTH_CLIENT_ID", "QBYgku9TlM8nF1yKGCMJzP0uofnsE2Sx")
var tenantURL = getEnvOrDefault("TUPLECTL_AUTH_BASE_URL", "https://dev-ak43b46u.eu.auth0.com")

var jwt = ""

func tryReadKeychain() bool {
	// get password
	secret, err := keyring.Get("Tuplestream", "default")
	if err != nil {
		return false
	}
	jwt = secret
	return false
}

func auth() {
	if tryReadKeychain() {
		return
	}

	form := url.Values{}
	form.Add("client_id", clientID)
	form.Add("scope", "logstream")
	form.Add("audience", "https://api.tuplestream.net/")

	resp, err := http.PostForm(tenantURL+"/oauth/device/code", form)
	handleError(err)
	defer resp.Body.Close()

	var cr CodeResponse
	err = json.NewDecoder(resp.Body).Decode(&cr)
	handleError(err)

	expiryDeadline := time.Now().Add(time.Second * time.Duration(cr.Expires))
	delayInterval := time.Duration(cr.Interval) * time.Second

	debug(fmt.Sprintf("Device code response status: %s", resp.Status))
	debug(fmt.Sprintf("Auth API callback URL: %s", cr.VerificationUriComplete))

	openbrowser(cr.VerificationUriComplete)

	for {
		debug(fmt.Sprintf("Sleeping for %v before polling again", delayInterval))
		time.Sleep(delayInterval)
		if time.Now().After(expiryDeadline) {
			log.Fatal("Couldn't verify device token in time")
		}

		pollForm := url.Values{}
		pollForm.Add("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
		pollForm.Add("client_id", clientID)
		pollForm.Add("device_code", cr.DeviceCode)

		resp, err = http.PostForm(tenantURL+"/oauth/token", pollForm)
		handleError(err)
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			var jwtError JwtError
			err = json.NewDecoder(resp.Body).Decode(&jwtError)
			handleError(err)

			debug(jwtError.Error)

			if jwtError.Error != "authorization_pending" {
				log.Fatal("Auth failed, came back with " + jwtError.Error)
			}
		} else {
			var success JwtSuccess
			err = json.NewDecoder(resp.Body).Decode(&success)
			handleError(err)
			print("Finished authentication! " + success.AccessToken)
			print("Type " + success.TokenType)
			print("expires in" + strconv.Itoa(success.ExpiresIn))
			jwt = success.AccessToken

			// set password in keyring
			err := keyring.Set("Tuplestream", "default", jwt)
			if err != nil {
				warn("unable to store credentials in the system keychain. " +
					"You'll have to repeat this process next time you run an authenticated tuplectl command")
			}
			break
		}
	}
}
