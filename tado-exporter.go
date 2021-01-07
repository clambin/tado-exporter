package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

func main() {
	authenticate2("christophe.lambin@gmail.com", "3nq7Y@cZ7Mpg")
}

func authenticate2(username, password string) {
	form := url.Values{}
	form.Add("client_id", "tado-web-app")
	form.Add("client_secret", "wZaRN7rpjn3FoNyF5IFuxg9uMzYJcvOoQ8QWiIqS3hfk6gLhVlG57j5YNoZL2Rtc")
	form.Add("grant_type", "password")
	form.Add("password", password)
	form.Add("scope", "home.user")
	form.Add("username", username)

	req, _ := http.NewRequest("POST", "https://auth.tado.com/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Add("Referer", "https://my.tado.com/")
	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	client := http.Client{}
	resp, err := client.Do(req)

	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			body, _ := ioutil.ReadAll(resp.Body)

			var resp interface{}
			jsonErr := json.Unmarshal(body, &resp)
			m := resp.(map[string]interface{})
			if jsonErr != nil {
				log.Fatal(jsonErr)
			}

			fmt.Println(m)
		}
		err = errors.New(resp.Status)
	}
	fmt.Println(err)
}
