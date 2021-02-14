package slackbot

import (
	"encoding/json"
	"fmt"
	"golang.org/x/net/websocket"
	"io/ioutil"
	"net/http"
	"sync/atomic"
)

type SlackBot struct {
	// counter needs to be first, otherwise atomic.AddUint64() may panic in 32 architectures
	// https://github.com/golang/go/issues/23345
	counter uint64
	ws      *websocket.Conn
	id      string
}

// Connect to slack & return a slackbot handle
func Connect(token string) (slackbot *SlackBot, err error) {
	var (
		wsURL string
		id    string
		ws    *websocket.Conn
	)
	if wsURL, id, err = slackStart(token); err == nil {
		if ws, err = websocket.Dial(wsURL, "", "https://api.slack.com/"); err == nil {
			slackbot = &SlackBot{ws: ws, id: id}
		}
	}
	return
}

// Message structure to send/receive messages to/from Slack
//
// These are the messages read off and written into the websocket. Since this
// struct serves as both read and write, we include the "Id" field which is
// required only for writing.
type Message struct {
	Id      uint64 `json:"id"`
	Type    string `json:"type"`
	Channel string `json:"channel"`
	Text    string `json:"text"`
}

// GetMessage waits for a message to be posted to the channel
func (bot *SlackBot) GetMessage() (m Message, err error) {
	err = websocket.JSON.Receive(bot.ws, &m)
	return
}

// PostMessage posts a new message to the channel
func (bot *SlackBot) PostMessage(m Message) error {
	m.Id = atomic.AddUint64(&bot.counter, 1)
	return websocket.JSON.Send(bot.ws, m)
}

type responseRtmStart struct {
	Ok    bool         `json:"ok"`
	Error string       `json:"error"`
	Url   string       `json:"url"`
	Self  responseSelf `json:"self"`
}

type responseSelf struct {
	Id string `json:"id"`
}

func slackStart(token string) (wsURL, id string, err error) {
	url := fmt.Sprintf("https://slack.com/api/rtm.start?token=%s", token)

	var resp *http.Response
	if resp, err = http.Get(url); err == nil {
		if resp.StatusCode == 200 {
			defer resp.Body.Close()

			var body []byte
			if body, err = ioutil.ReadAll(resp.Body); err == nil {
				var respObj responseRtmStart
				if err = json.Unmarshal(body, &respObj); err == nil {
					if respObj.Ok {
						wsURL = respObj.Url
						id = respObj.Self.Id

					} else {
						err = fmt.Errorf("slack error: %s", respObj.Error)
					}
				} else {
					err = fmt.Errorf("invalid response received: %s", err.Error())
				}
			}
		} else {
			err = fmt.Errorf("API request failed:  %d - %s", resp.StatusCode, resp.Status)
		}
	} else {
		err = fmt.Errorf("API request failed: %s", err.Error())
	}

	return
}
