package main

import (
	"encoding/json"
	"fmt"
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

func (n *Notifier) notifyToSlack(action, key, value string) error {
	payload, err := createSlackJsonMessage(
		fmt.Sprintf("*%s* event occurred!!!", action),
		SlackAttachment{Title: key, Text: value},
	)
	if err != nil {
		return err
	}

	for _, u := range n.incomingWebhookUrls.Load().([]string) {
		res, err := http.PostForm(u, url.Values{"payload": {string(payload)}})
		if err != nil {
			return err
		}
		res.Body.Close()
		if res.StatusCode != 200 {
			return fmt.Errorf("slack post failed. status code = %d", res.StatusCode)
		}
	}

	return nil
}

func createSlackJsonMessage(text string, attachments ...SlackAttachment) ([]byte, error) {
	return json.Marshal(SlackMessage{
		Text:        text,
		Attachments: attachments,
	})
}
