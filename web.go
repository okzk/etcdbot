package main

import (
	"fmt"
	"github.com/coreos/etcd/client"
	"golang.org/x/net/context"
	"log"
	"net/http"
	"regexp"
	"strings"
)

type MyMux struct {
	token       string
	channelName string
	triggerWord string
	keysApi     client.KeysAPI
	keys        []string
}

func (m *MyMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" || r.Method == "HEAD" {
		return
	}
	if r.Method != "POST" {
		w.WriteHeader(http.StatusNotImplemented)
		return
	}

	r.ParseForm()

	if m.token != "" && m.token != r.PostForm.Get("token") {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if m.channelName != "" && m.channelName != r.PostForm.Get("channel_name") {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if r.PostForm.Get("user_name") == "slackbot" {
		return
	}

	text := r.PostForm.Get("text")
	if !strings.HasPrefix(text, m.triggerWord) {
		return
	}

	args := strings.TrimSpace(strings.TrimPrefix(text, m.triggerWord))
	if args == "get" {
		attachment := []SlackAttachment{}
		for _, k := range m.keys {
			res, err := m.keysApi.Get(context.Background(), k, nil)
			if err != nil {
				if e, b := err.(client.Error); b && e.Code == client.ErrorCodeKeyNotFound {
					continue
				}
				writeJsonResponse(w, "internal error occurred...")
				return
			}
			attachment = append(attachment, SlackAttachment{Title: k, Text: res.Node.Value})
		}

		writeJsonResponse(w, fmt.Sprintf("%d key(s) exists...", len(attachment)), attachment...)
		return
	}

	p := regexp.MustCompile(`^delete\s+([/\w\-\.]+)$`)
	if g := p.FindStringSubmatch(args); len(g) == 2 {
		key := g[1]
		if m.hasKey(key) {
			m.keysApi.Delete(context.Background(), key, nil)
			writeJsonResponse(w, "accepted!")
		} else {
			writeJsonResponse(w, "?")
		}
		return
	}
	writeJsonResponse(w, "?")
}

func writeJsonResponse(w http.ResponseWriter, text string, attachments ...SlackAttachment) {
	json, err := createSlackJsonMessage(text, attachments...)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

func (m *MyMux) hasKey(key string) bool {
	for _, k := range m.keys {
		if k == key {
			return true
		}
	}
	return false
}

func listenAndServe(config *AppConfig, keysApi client.KeysAPI) {
	mux := MyMux{
		token:       config.Slack.Hook.Token,
		channelName: config.Slack.Hook.ChannelName,
		triggerWord: config.Slack.Hook.TriggerWord,
		keysApi:     keysApi,
		keys:        config.Etcd.WatchTargets,
	}

	log.Println("[INFO] Start outgoing hook server...")
	err := http.ListenAndServe(fmt.Sprintf(":%d", config.Slack.Hook.Port), &mux)
	log.Fatal("[FATAL] ", err)
}
