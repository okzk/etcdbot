package main

import (
	"encoding/json"
	"fmt"
	log "github.com/cihub/seelog"
	"net/http"
	"net/url"
)

type SlackAttachment struct {
	Title string `json:"title,omitempty"`
	Text  string `json:"text,omitempty"`
}

type SlackMessage struct {
	Text        string            `json:"text,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
}

func (nc *NotifierConfig) notifyToSlack(action, key, value string) (err error) {
	payload, err := createSlackJsonMessage(
		fmt.Sprintf("A _*%s*_ event occurred!!!", action),
		SlackAttachment{Title: key, Text: value},
	)
	if err != nil {
		return
	}

	if nc.dryRun {
		log.Debug("Dry run mode! payload: ", string(payload))
		return
	}

	res, err := http.PostForm(
		nc.incomingWebhookUrl,
		url.Values{"payload": {string(payload)}},
	)
	if err != nil {
		return
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		err = fmt.Errorf("slack post failed. status code = %d", res.StatusCode)
	}

	return
}

func createSlackJsonMessage(text string, attachments ...SlackAttachment) ([]byte, error) {
	return json.Marshal(SlackMessage{
		Text:        text,
		Attachments: attachments,
	})
}
