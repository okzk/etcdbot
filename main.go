package main

import (
	"flag"
	log "github.com/cihub/seelog"
	"golang.org/x/net/context"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	defer log.Flush()
	flag.Parse()
	err := initLogger()
	if err != nil {
		log.Critical(err)
		return
	}

	cfg := loadAppConfig()
	nc := createNotifierConfig(cfg)

	if cfg.Slack.Hook.Enable {
		go listenAndServe(cfg, nc.keysApi)
	}

	ctx, cancel := context.WithCancel(context.Background())

	log.Info("Start watching keys...")
	wg := sync.WaitGroup{}
	for _, key := range cfg.Etcd.WatchTargets {
		wg.Add(1)
		go func(key string) {
			nc.watchAndNotify(ctx, key)
			wg.Done()
		}(key)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	sig := <-sigCh
	log.Infof("Signal(%v) recieved", sig)
	log.Info("Closing all watchers...")
	cancel()
	wg.Wait()
	log.Info("Now shutting down.")
}
