package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
)

type SlackPostAttachment struct {
	Title string `json:"title,omitempty"`
	Text  string `json:"text,omitempty"`
}

type SlackPostPayload struct {
	Text        string                `json:"text,omitempty"`
	Attachments []SlackPostAttachment `json:"attachments,omitempty"`
}

func (nc *NotifierConfig) notifyToSlack(action, key, value string) (err error) {
	payload, err := json.Marshal(
		SlackPostPayload{
			Text: fmt.Sprintf("A %s event occurred!", action),
			Attachments: []SlackPostAttachment{
				SlackPostAttachment{
					Title: key,
					Text:  value,
				},
			},
		})
	if err != nil {
		return
	}

	if nc.dryRun {
		log.Println("[DEBUG] payload: ", string(payload))
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
