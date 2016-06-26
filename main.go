package main

import (
	"flag"
	"golang.org/x/net/context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	flag.Parse()
	cfg := loadAppConfig()
	nc := createNotifierConfig(cfg)

	ctx, cancel := context.WithCancel(context.Background())

	log.Println("[INFO] Start watching keys...")
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
	log.Printf("[INFO] Signal(%v) recieved", sig)
	log.Println("[INFO] Closing all watchers...")
	cancel()
	wg.Wait()
	log.Printf("[INFO] Now shutting down.")
}
