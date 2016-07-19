package main

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/coreos/etcd/client"
	"github.com/nlopes/slack"
	"golang.org/x/net/context"
	"path"
	"strings"
)

type Config struct {
	rtm     *slack.RTM
	keysApi client.KeysAPI
	metaDir string
}

func (c *Config) run(channel string, args []string) {
	if len(args) == 0 {
		c.runHelp(channel)
		return
	}

	switch args[0] {
	case "get":
		c.runGet(channel, args[1:])
	case "delete":
		c.runDelete(channel, args[1:])
	case "help":
		c.runHelp(channel)
	default:
		c.runHelp(channel)
	}
}

func (c *Config) runGet(channel string, args []string) {
	if len(args) != 0 {
		c.runHelp(channel)
		return
	}
	res, err := c.keysApi.Get(context.Background(), path.Join(c.metaDir, "watchTargetList"), &client.GetOptions{})
	if err != nil {
		if e, ok := err.(client.Error); ok && e.Code == client.ErrorCodeKeyNotFound {
			c.rtm.PostMessage(channel, "not key exists.", slack.NewPostMessageParameters())
		} else {
			log.Error(err)
			c.rtm.PostMessage(channel, "internal error.", slack.NewPostMessageParameters())
		}
		return
	}

	attachments := []slack.Attachment{}
	for _, key := range strings.Split(res.Node.Value, ",") {
		res, err = c.keysApi.Get(context.Background(), key, &client.GetOptions{})
		if err != nil {
			if e, ok := err.(client.Error); ok && e.Code == client.ErrorCodeKeyNotFound {
				continue
			}
			log.Error(err)
			c.rtm.PostMessage(channel, "internal error.", slack.NewPostMessageParameters())
			return
		}
		attachments = append(attachments, slack.Attachment{Title: key, Text: res.Node.Value})
	}
	if len(attachments) == 0 {
		c.rtm.PostMessage(channel, "not key exists.", slack.NewPostMessageParameters())
	} else {
		params := slack.NewPostMessageParameters()
		params.Attachments = attachments
		c.rtm.PostMessage(channel, fmt.Sprintf("%d key(s) exists...", len(attachments)), params)
	}
}

func (c *Config) runDelete(channel string, args []string) {
	if len(args) != 1 {
		c.runHelp(channel)
		return
	}
	key := args[0]

	res, err := c.keysApi.Get(context.Background(), path.Join(c.metaDir, "watchTargetList"), &client.GetOptions{})
	if err != nil {
		if e, ok := err.(client.Error); ok && e.Code == client.ErrorCodeKeyNotFound {
			c.rtm.PostMessage(channel, "not in watch list", slack.NewPostMessageParameters())
		} else {
			log.Error(err)
			c.rtm.PostMessage(channel, "internal error.", slack.NewPostMessageParameters())
		}
		return
	}
	if !existsInSlice(key, strings.Split(res.Node.Value, ",")) {
		c.rtm.PostMessage(channel, "not in watch list", slack.NewPostMessageParameters())
		return
	}

	log.Info("Deleting... ", key)
	res, err = c.keysApi.Delete(context.Background(), key, &client.DeleteOptions{})
	if err != nil {
		if e, ok := err.(client.Error); ok && e.Code == client.ErrorCodeKeyNotFound {
			c.rtm.PostMessage(channel, "not exists!", slack.NewPostMessageParameters())
		} else {
			log.Error(err)
			c.rtm.PostMessage(channel, "internal error.", slack.NewPostMessageParameters())
		}
		return
	}
	c.rtm.PostMessage(channel, "OK", slack.NewPostMessageParameters())
}

func (c *Config) runHelp(channel string) {
	c.rtm.PostMessage(channel, "?", slack.NewPostMessageParameters())
}

func existsInSlice(s string, list []string) bool {
	for _, a := range list {
		if s == a {
			return true
		}
	}
	return false
}
