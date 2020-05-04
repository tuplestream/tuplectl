package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"time"
)

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
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}

}

func handleError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

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
}

var clientID = "QBYgku9TlM8nF1yKGCMJzP0uofnsE2Sx"
var tenantURL = "https://dev-ak43b46u.eu.auth0.com"

func main() {
	// verbose := flag.String("v", "", "Get the current version of tuplectl")
	// flag.Parse()

	// if flag.NArg() == 0 {
	// 	flag.Usage()
	// 	os.Exit(1)
	// }
	// cmd := os.Args[0]
	// if cmd == "" {
	// 	flag.Usage()
	// 	os.Exit(1)
	// }

	form := url.Values{}
	form.Add("client_id", clientID)
	form.Add("scope", "read:email")
	form.Add("audience", "https://api.tuplestream.net/")

	resp, err := http.PostForm(tenantURL+"/oauth/device/code", form)
	handleError(err)
	defer resp.Body.Close()

	var cr CodeResponse
	err = json.NewDecoder(resp.Body).Decode(&cr)
	handleError(err)

	expiryDeadline := time.Now().Add(time.Second * time.Duration(cr.Expires))
	delayInterval := time.Duration(cr.Interval) * time.Second

	log.Print(resp.Status)
	log.Print(cr.VerificationUriComplete)

	openbrowser(cr.VerificationUriComplete)

	for {
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

			log.Print(jwtError.Error)

			if jwtError.Error != "authorization_pending" {
				log.Panic("Auth failed, came back with " + jwtError.Error)
			}
		} else {
			var success JwtSuccess
			err = json.NewDecoder(resp.Body).Decode(&success)
			handleError(err)
			print("Finished authentication! " + success.AccessToken)
			print("Type " + success.TokenType)
			break
		}
	}
}
