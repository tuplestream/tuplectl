package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/zalando/go-keyring"
)

// types
type codeResponse struct {
	VerificationURIComplete string `json:"verification_uri_complete"`
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	Interval                int    `json:"interval"`
	ExpiresIn               int    `json:"expires_in"`
}

type jwtError struct {
	Error string `json:"error"`
}

type jwtSuccess struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

type jsonWebKeys struct {
	Kty string   `json:"kty"`
	Kid string   `json:"kid"`
	Use string   `json:"use"`
	N   string   `json:"n"`
	E   string   `json:"e"`
	X5c []string `json:"x5c"`
}

type jwks struct {
	Keys []jsonWebKeys `json:"keys"`
}

func printAuthAddress(url string) {
	fmt.Println("To continue authentication, open this url in a browser: " + url)
}

func waitForKeyPress() {
	reader := bufio.NewReader(os.Stdin)
	_, err := reader.ReadString('\n')
	handleError(err)
}

func openbrowser(url string) {
	var err error

	if getEnvOrDefault("TUPLECTL_OPEN_BROWSER", "true") != "true" {
		printAuthAddress(url)
		return
	}

	switch runtime.GOOS {
	case "linux":
		waitForKeyPress()
		err = exec.Command("xdg-open", url).Start()
	case "darwin":
		waitForKeyPress()
		err = exec.Command("open", url).Start()
	default:
		printAuthAddress(url)
	}

	handleError(err)
}

var authKeyName = "com.tuplestream.tuplectl.AccessToken"
var keychainUser = "default"
var DefaultClientID string
var DefaultTenantURL string
var clientID = getEnvOrDefault("TUPLECTL_AUTH_CLIENT_ID", DefaultClientID)
var tenantURL = getEnvOrDefault("TUPLECTL_AUTH_BASE_URL", DefaultTenantURL)

var accessToken = ""

// from https://auth0.com/docs/quickstart/backend/golang/01-authorization#validate-access-tokens
func getPemCert(token *jwt.Token) (string, error) {
	cert := ""
	resp, err := http.Get(tenantURL + "/.well-known/jwks.json")

	if err != nil {
		return cert, err
	}
	defer resp.Body.Close()

	var jwks = jwks{}
	err = json.NewDecoder(resp.Body).Decode(&jwks)

	if err != nil {
		return cert, err
	}

	for k := range jwks.Keys {
		if token.Header["kid"] == jwks.Keys[k].Kid {
			cert = "-----BEGIN CERTIFICATE-----\n" + jwks.Keys[k].X5c[0] + "\n-----END CERTIFICATE-----"
		}
	}

	if cert == "" {
		err := errors.New("unable to find appropriate key")
		return cert, err
	}

	return cert, nil
}

type customClaims struct {
	Scope string `json:"scope"`
	jwt.StandardClaims
}

// attempt to pull a jwt from the system key store
// if a valid, in-date jwt is found, returns true
// in all other circumstances returns false
func tryReadKeychain() bool {
	// get raw token string
	secret, err := keyring.Get(authKeyName, keychainUser)
	if err != nil || secret == "" {
		return false
	}

	token, _ := jwt.ParseWithClaims(secret, &customClaims{}, func(token *jwt.Token) (interface{}, error) {
		cert, err := getPemCert(token)
		if err != nil {
			return nil, err
		}
		result, _ := jwt.ParseRSAPublicKeyFromPEM([]byte(cert))
		return result, nil
	})

	if !token.Valid {
		keyring.Delete(authKeyName, keychainUser)
		return false
	}

	accessToken = secret
	return true
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

	var cr codeResponse
	err = json.NewDecoder(resp.Body).Decode(&cr)
	handleError(err)

	expiryDeadline := time.Now().Add(time.Second * time.Duration(cr.ExpiresIn))
	delayInterval := time.Duration(cr.Interval) * time.Second

	debug(fmt.Sprintf("Device code response status: %s", resp.Status))
	debug(fmt.Sprintf("Auth API callback URL: %s", cr.VerificationURIComplete))

	// tell user we're about to open a browser window, give them the code to look out for
	fmt.Println(fmt.Sprintf("We need to authenticate you through a browser. Verify code shown is %s", red(cr.UserCode)))
	fmt.Println("Press any key to start")
	openbrowser(cr.VerificationURIComplete)

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
			var jwtError jwtError
			err = json.NewDecoder(resp.Body).Decode(&jwtError)
			handleError(err)

			debug(jwtError.Error)

			if jwtError.Error != "authorization_pending" {
				log.Fatal("Auth failed, came back with " + jwtError.Error)
			}
		} else {
			var success jwtSuccess
			err = json.NewDecoder(resp.Body).Decode(&success)
			handleError(err)
			print("Finished authentication!")

			if getEnvOrDefault("TUPLECTL_PRINT_AUTH_TOKEN", "") != "" {
				print(success.AccessToken)
			}

			// set password in keyring
			err := keyring.Set(authKeyName, keychainUser, success.AccessToken)
			if err != nil {
				warn("unable to store credentials in the system keychain. " +
					"You'll have to repeat this process next time you run an authenticated tuplectl command")
			}
			break
		}
	}
}
