package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	"github.com/zalando/go-keyring"
)

type CodeResponse struct {
	VerificationUriComplete string `json:"verification_uri_complete"`
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	Interval                int    `json:"interval"`
	ExpiresIn               int    `json:"expires_in"`
}

type JwtError struct {
	Error string `json:"error"`
}

type JwtSuccess struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

func openbrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
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

// attempt to pull a jwt from the system key store
// if a valid, in-date jwt is found, returns true
// in all other circumstances returns false
func tryReadKeychain() bool {
	// get password
	secret, err := keyring.Get("Tuplestream", "default")
	if err != nil {
		return false
	}
	jwt = secret
	return false
}

func keychainString(data JwtSuccess) string {
	jwtExpiry := time.Now().Add(time.Duration(data.ExpiresIn) * time.Second)
	bytes, err := jwtExpiry.MarshalText()
	handleError(err)
	return string(bytes) + "|" + data.AccessToken
}

func doAuth() {
	// see if we have a current valid token already
	if tryReadKeychain() {
		return
	}

	// initiate auth, ask for device token
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

	expiryDeadline := time.Now().Add(time.Second * time.Duration(cr.ExpiresIn))
	delayInterval := time.Duration(cr.Interval) * time.Second

	debug(fmt.Sprintf("Device code response status: %s", resp.Status))
	debug(fmt.Sprintf("Auth API callback URL: %s", cr.VerificationUriComplete))

	// tell user we're about to open a browser window, give them the code to look out for
	fmt.Println(fmt.Sprintf("We need to authenticate you through a browser. Verify code shown is %s", red(cr.UserCode)))
	fmt.Println("Press any key to start")
	reader := bufio.NewReader(os.Stdin)
	_, err = reader.ReadString('\n')
	handleError(err)
	openbrowser(cr.VerificationUriComplete)

	for {
		// wait for a token- continue polling while user is doing their
		// thing in the browser
		debug(fmt.Sprintf("Sleeping for %v before polling again", delayInterval))
		time.Sleep(delayInterval)

		// user took to long or abandonded auth- give up
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
