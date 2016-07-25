package main

import (
	"flag"
	log "github.com/cihub/seelog"
	"github.com/coreos/etcd/client"
	"github.com/nlopes/slack"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	defer log.Flush()
	flag.Parse()
	err := initLogger()
	if err != nil {
		log.Critical(err)
		return
	}

	serveBot()
	log.Info("Now shutting down.")
}

func serveBot() {
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
	watchBase := os.Getenv("BOT_WATCH_TARGET_BASE")
	if watchBase == "" {
		watchBase = "/public/"
	}

	keysApi := client.NewKeysAPI(c)

	key := os.Getenv("BOT_SLACK_API_KEY")
	if key == "" {
		log.Critical("Missing slack api key!")
		os.Exit(1)
	}

	api := slack.New(key)
	auth, err := api.AuthTest()
	if err != nil {
		log.Critical(err)
		os.Exit(1)
	}
	id := auth.UserID

	rtm := api.NewRTM()
	defer rtm.Disconnect()
	mngCh := make(chan int, 1)
	go func() {
		rtm.ManageConnection()
		close(mngCh)
	}()

	conf := &Config{rtm: rtm, keysApi: keysApi, metaDir: metaDir, watchBase: watchBase}

	log.Info("Starting RTM loop...")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.ConnectedEvent:
				log.Info("RTM connected...")
			case *slack.MessageEvent:
				if strings.HasPrefix(ev.Text, "<@"+id+">") {
					log.Info("User: ", ev.User, ", Channel: ", ev.Channel, ", Text: ", ev.Text)
					conf.run(ev.Channel, strings.Fields(ev.Text)[1:])
				}
			case *slack.InvalidAuthEvent:
				log.Error("Invalid credentials")
				return
			case *slack.DisconnectedEvent:
				if ev.Intentional {
					log.Info("RTM connection intentionally closed.")
					return
				}
			}
		case <-mngCh:
			log.Error("ManageConnection goroutine unexpectedly finished!!!")
			return
		case sig := <-sigCh:
			log.Infof("Signal(%v) recieved", sig)
			return
		}
	}
}
