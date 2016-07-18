package main

import (
	"flag"
	log "github.com/cihub/seelog"
	"os"
	"os/signal"
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

	n := NewNotifier()

	log.Info("Start watching keys...")
	n.Start()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	sig := <-sigCh
	log.Infof("Signal(%v) recieved", sig)
	log.Info("Closing all watchers...")
	n.Stop()
	log.Info("Now shutting down.")
}
