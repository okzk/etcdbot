package main

import (
	"bytes"
	"context"
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/coreos/etcd/client"
	"github.com/nlopes/slack"
	"path"
	"regexp"
	"strings"
)

var (
	ctx        context.Context = context.Background()
	pathRegexp *regexp.Regexp  = regexp.MustCompile(`^(/[\w\-\.]+)+$`)
)

type Config struct {
	rtm     *slack.RTM
	keysApi client.KeysAPI
	metaDir string

	watchBase string
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
	case "watch":
		c.runWatch(channel, args[1:])
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
	res, err := c.keysApi.Get(ctx, c.watchListKey(), &client.GetOptions{})
	if err != nil {
		if e, ok := err.(client.Error); ok && e.Code == client.ErrorCodeKeyNotFound {
			c.rtm.PostMessage(channel, "no entry exists.", slack.NewPostMessageParameters())
		} else {
			log.Error(err)
			c.rtm.PostMessage(channel, "internal error.", slack.NewPostMessageParameters())
		}
		return
	}

	attachments := []slack.Attachment{}
	for _, key := range strings.Split(res.Node.Value, ",") {
		res, err = c.keysApi.Get(ctx, key, &client.GetOptions{})
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
		c.rtm.PostMessage(channel, "no entry exists.", slack.NewPostMessageParameters())
	} else {
		params := slack.NewPostMessageParameters()
		params.Attachments = attachments
		c.rtm.PostMessage(channel, fmt.Sprintf("entry count: %d", len(attachments)), params)
	}
}

func (c *Config) runDelete(channel string, args []string) {
	if len(args) != 1 {
		c.runHelp(channel)
		return
	}
	key := args[0]

	res, err := c.keysApi.Get(ctx, c.watchListKey(), &client.GetOptions{})
	if err != nil {
		if e, ok := err.(client.Error); ok && e.Code == client.ErrorCodeKeyNotFound {
			c.rtm.PostMessage(channel, "not in watch list.", slack.NewPostMessageParameters())
		} else {
			log.Error(err)
			c.rtm.PostMessage(channel, "internal error.", slack.NewPostMessageParameters())
		}
		return
	}
	if !existsInSlice(key, strings.Split(res.Node.Value, ",")) {
		c.rtm.PostMessage(channel, "not in watch list.", slack.NewPostMessageParameters())
		return
	}

	log.Info("Deleting... ", key)
	_, err = c.keysApi.Delete(ctx, key, &client.DeleteOptions{})
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

func (c *Config) runWatch(channel string, args []string) {
	if len(args) == 0 {
		c.runHelp(channel)
		return
	}

	switch args[0] {
	case "list":
		c.runWatchList(channel, args[1:])
	case "add":
		c.runWatchAdd(channel, args[1:])
	case "delete":
		c.runWatchDelete(channel, args[1:])
	default:
		c.runHelp(channel)
	}
}

func (c *Config) runWatchList(channel string, args []string) {
	if len(args) != 0 {
		c.runHelp(channel)
		return
	}
	res, err := c.keysApi.Get(ctx, c.watchListKey(), &client.GetOptions{})
	if err != nil {
		if e, ok := err.(client.Error); ok && e.Code == client.ErrorCodeKeyNotFound {
			c.rtm.PostMessage(channel, "empty watch list.", slack.NewPostMessageParameters())
		} else {
			log.Error(err)
			c.rtm.PostMessage(channel, "internal error.", slack.NewPostMessageParameters())
		}
		return
	}

	var buffer bytes.Buffer
	for _, k := range strings.Split(res.Node.Value, ",") {
		buffer.WriteString("- ")
		buffer.WriteString(k)
		buffer.WriteRune('\n')
	}
	c.rtm.PostMessage(channel, buffer.String(), slack.NewPostMessageParameters())
}

func (c *Config) runWatchAdd(channel string, args []string) {
	if len(args) != 1 {
		c.runHelp(channel)
		return
	}
	key := args[0]
	if !strings.HasPrefix(key, c.watchBase) || !pathRegexp.MatchString(key) {
		c.rtm.PostMessage(channel, "invalid key!", slack.NewPostMessageParameters())
		return
	}

	keys := []string{}
	setOpts := client.SetOptions{}
	res, err := c.keysApi.Get(ctx, c.watchListKey(), &client.GetOptions{})
	if err != nil {
		if e, ok := err.(client.Error); ok && e.Code == client.ErrorCodeKeyNotFound {
			setOpts.PrevExist = client.PrevNoExist
		} else {
			log.Error(err)
			c.rtm.PostMessage(channel, "internal error.", slack.NewPostMessageParameters())
			return
		}
	} else {
		keys = strings.Split(res.Node.Value, ",")
		setOpts.PrevIndex = res.Node.ModifiedIndex
	}

	if existsInSlice(key, keys) {
		c.rtm.PostMessage(channel, "already in watch list.", slack.NewPostMessageParameters())
		return
	}

	log.Info("Updating watch key list... ")
	_, err = c.keysApi.Set(ctx, c.watchListKey(), strings.Join(append(keys, key), ","), &setOpts)
	if err != nil {
		log.Error(err)
		c.rtm.PostMessage(channel, "internal error.", slack.NewPostMessageParameters())
	} else {
		c.rtm.PostMessage(channel, "OK", slack.NewPostMessageParameters())
	}
}

func (c *Config) runWatchDelete(channel string, args []string) {
	if len(args) != 1 {
		c.runHelp(channel)
		return
	}
	key := args[0]

	res, err := c.keysApi.Get(ctx, c.watchListKey(), &client.GetOptions{})
	if err != nil {
		if e, ok := err.(client.Error); ok && e.Code == client.ErrorCodeKeyNotFound {
			c.rtm.PostMessage(channel, "not in watch list.", slack.NewPostMessageParameters())
		} else {
			log.Error(err)
			c.rtm.PostMessage(channel, "internal error.", slack.NewPostMessageParameters())
		}
		return
	}

	keys := strings.Split(res.Node.Value, ",")
	if !existsInSlice(key, keys) {
		c.rtm.PostMessage(channel, "not in watch list.", slack.NewPostMessageParameters())
		return
	}

	log.Info("Updating watch key list... ")
	newKeys := deleteFromSlice(key, keys)
	if len(newKeys) == 0 {
		_, err = c.keysApi.Delete(ctx, c.watchListKey(), &client.DeleteOptions{PrevIndex: res.Node.ModifiedIndex})
	} else {
		_, err = c.keysApi.Set(ctx, c.watchListKey(), strings.Join(newKeys, ","), &client.SetOptions{PrevIndex: res.Node.ModifiedIndex})
	}
	if err != nil {
		log.Error(err)
		c.rtm.PostMessage(channel, "internal error.", slack.NewPostMessageParameters())
	} else {
		c.rtm.PostMessage(channel, "OK", slack.NewPostMessageParameters())
	}
}

func (c *Config) runHelp(channel string) {
	usage := `command list:
- get
- delete PATH
- watch list
- watch add PATH
- watch delete PATH
`
	c.rtm.PostMessage(channel, usage, slack.NewPostMessageParameters())
}

func (c *Config) watchListKey() string {
	return path.Join(c.metaDir, "watchTargetList")
}

func existsInSlice(s string, list []string) bool {
	for _, a := range list {
		if s == a {
			return true
		}
	}
	return false
}

func deleteFromSlice(s string, list []string) []string {
	ret := make([]string, 0, len(list))
	for _, a := range list {
		if a != s {
			ret = append(ret, a)
		}
	}
	return ret
}
