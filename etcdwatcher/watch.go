package main

import (
	log "github.com/cihub/seelog"
	"github.com/coreos/etcd/client"
	"golang.org/x/net/context"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Notifier struct {
	metaDir             string
	keysApi             client.KeysAPI
	incomingWebhookUrls atomic.Value

	ctx    context.Context
	cancel context.CancelFunc

	wg sync.WaitGroup
}

func NewNotifier() *Notifier {
	endpoints := os.Getenv("BOT_ETCD_ENDPOINTS")
	if endpoints == "" {
		endpoints = "http://localhost:2379"
	}

	c, err := client.New(client.Config{
		Endpoints:               strings.Split(endpoints, ","),
		Transport:               client.DefaultTransport,
		Username:                os.Getenv("BOT_ETCD_USER"),
		Password:                os.Getenv("BOT_ETCD_PASSWORD"),
		HeaderTimeoutPerRequest: time.Second,
	})
	if err != nil {
		log.Critical(err)
		os.Exit(1)
	}

	metaDir := os.Getenv("BOT_METADATA_DIR")
	if metaDir == "" {
		metaDir = "/etcdbot_meta/"
	}

	ctx, cancel := context.WithCancel(context.Background())

	n := &Notifier{
		metaDir: metaDir,
		keysApi: client.NewKeysAPI(c),

		ctx:    ctx,
		cancel: cancel,
	}
	n.incomingWebhookUrls.Store([]string{})
	return n
}

func (n *Notifier) Start() {
	n.wg.Add(2)

	go n.watchIncomingWebHookUrls()
	go n.watchTargetList()
}

func (nc *Notifier) Stop() {
	nc.cancel()
	nc.wg.Wait()
}

func (nc *Notifier) watchIncomingWebHookUrls() {
	defer nc.wg.Done()

	key := path.Join(nc.metaDir, "incomingWebHookUrls")

	var index uint64
	res, err := nc.keysApi.Get(nc.ctx, key, &client.GetOptions{})
	if err != nil {
		log.Error("Fail to initialize incomingWebHookUrls.")
		log.Error(err)
		if e, ok := err.(client.Error); ok {
			index = e.Index
		}
	} else {
		index = res.Index + 1
		if res.Node.Value != "" {
			nc.incomingWebhookUrls.Store(strings.Split(res.Node.Value, ","))
		}
	}

	watchOp := client.WatcherOptions{AfterIndex: index}
	watcher := nc.keysApi.Watcher(key, &watchOp)
	for {
		res, err := watcher.Next(nc.ctx)
		if err == context.Canceled {
			log.Debugf("Stop watching key:%s", key)
			return
		}
		if err != nil {
			log.Errorf("key=%s, err=%v", key, err)
			continue
		}
		log.Info("Updating incomingWebHookUrls...")
		if res.Node.Value == "" {
			nc.incomingWebhookUrls.Store([]string{})
		} else {
			nc.incomingWebhookUrls.Store(strings.Split(res.Node.Value, ","))
		}
	}
}

func (n *Notifier) watchTargetList() {
	defer n.wg.Done()

	key := path.Join(n.metaDir, "watchTargetList")
	ctx, cancel := context.WithCancel(n.ctx)

	var index uint64
	res, err := n.keysApi.Get(n.ctx, key, &client.GetOptions{})
	if err != nil {
		log.Error("Fail to initialize watch target list.")
		log.Error(err)
		if e, ok := err.(client.Error); ok {
			index = e.Index
		}
	} else if res.Node.Value != "" {
		index = res.Index + 1
		for _, w := range strings.Split(res.Node.Value, ",") {
			go n.watchAndNotify(ctx, w)
		}
	}

	watchOp := client.WatcherOptions{AfterIndex: index}
	watcher := n.keysApi.Watcher(key, &watchOp)
	for {
		res, err := watcher.Next(n.ctx)
		if err == context.Canceled {
			log.Debugf("Stop watching key:%s", key)
			return
		}
		if err != nil {
			log.Errorf("key=%s, err=%v", key, err)
			continue
		}

		cancel()
		ctx, cancel = context.WithCancel(n.ctx)
		newList := []string{}
		if res.Node.Value != "" {
			newList = strings.Split(res.Node.Value, ",")
		}
		log.Info("Updating watch target list... ", newList)
		for _, w := range newList {
			go n.watchAndNotify(ctx, w)
		}
	}
}

func (n *Notifier) watchAndNotify(ctx context.Context, key string) {
	n.wg.Add(1)
	defer n.wg.Done()

	log.Debugf("Start watching key:%s", key)
	watchOp := client.WatcherOptions{}
	setOp := client.SetOptions{PrevExist: client.PrevNoExist, TTL: time.Minute}
	watcher := n.keysApi.Watcher(key, &watchOp)
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

		_, err = n.keysApi.Set(context.Background(), path.Join(n.metaDir, "lock", strconv.FormatUint(res.Index, 10)), "lock", &setOp)
		if err != nil {
			log.Debugf("Faii to acquire a lock, skip %s notification. key=%s, err=%v", res.Action, key, err)
			continue
		}

		err = n.notifyToSlack(res.Action, res.Node.Key, res.Node.Value)
		if err != nil {
			log.Errorf("key=%s, err=%v", key, err)
		} else {
			log.Debugf("Notified! key=%s", key)
		}
	}
}
