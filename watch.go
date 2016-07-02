package main

import (
	log "github.com/cihub/seelog"
	"github.com/coreos/etcd/client"
	"golang.org/x/net/context"
	"os"
	"path"
	"strconv"
	"time"
)

type NotifierConfig struct {
	lockDir            string
	keysApi            client.KeysAPI
	incomingWebhookUrl string
	dryRun             bool
}

func createNotifierConfig(cfg *AppConfig) *NotifierConfig {
	c, err := client.New(client.Config{
		Endpoints:               cfg.Etcd.Endpoints,
		Transport:               client.DefaultTransport,
		HeaderTimeoutPerRequest: time.Second,
	})
	if err != nil {
		log.Critical(err)
		os.Exit(1)
	}

	return &NotifierConfig{
		lockDir:            cfg.Etcd.LockDir,
		keysApi:            client.NewKeysAPI(c),
		incomingWebhookUrl: cfg.Slack.IncomingWebhookUrl,
		dryRun:             cfg.Slack.DryRun,
	}
}

func (nc *NotifierConfig) watchAndNotify(ctx context.Context, key string) {
	log.Debugf("Start watching key:%s", key)
	watchOp := client.WatcherOptions{}
	setOp := client.SetOptions{PrevExist: client.PrevNoExist, TTL: time.Minute}
	watcher := nc.keysApi.Watcher(key, &watchOp)
	for {
		res, err := watcher.Next(ctx)
		if err == context.Canceled {
			log.Debugf("Stop watching key:%s", key)
			return
		}
		if err != nil {
			log.Errorf("key=%s, err=%v", key, err)
			time.Sleep(10 * time.Second)
			continue
		}
		if res.PrevNode != nil && res.Node.Value == res.PrevNode.Value {
			log.Debugf("Same value is set on %s, skip %s notification", key, res.Action)
			continue
		}

		_, err = nc.keysApi.Set(context.Background(), path.Join(nc.lockDir, strconv.FormatUint(res.Index, 10)), "lock", &setOp)
		if err != nil {
			log.Debugf("Faii to acquire a lock, skip %s notification. key=%s, err=%v", res.Action, key, err)
			continue
		}

		err = nc.notifyToSlack(res.Action, res.Node.Key, res.Node.Value)
		if err != nil {
			log.Errorf("key=%s, err=%v", key, err)
		} else {
			log.Debugf("Notified! key=%s", key)
		}

	}
}
