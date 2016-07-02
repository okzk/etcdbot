package main

import (
	"flag"
	log "github.com/cihub/seelog"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

var (
	appConfigFile string
	logXmlFile    string
)

func init() {
	flag.StringVar(&appConfigFile, "cfg", "", "config yaml file")
	flag.StringVar(&logXmlFile, "log", "", "log xml file")
}

type AppConfig struct {
	Etcd struct {
		Endpoints    []string `yaml:"endpoints"`
		WatchTargets []string `yaml:"watch_targets"`
		LockDir      string   `yaml:"lock_dir"`
	} `yaml:"etcd"`
	Slack struct {
		IncomingWebhookUrl string `yaml:"incoming_webhook_url"`
		DryRun             bool   `yaml:"dry_run"`

		Hook struct {
			Enable      bool   `yaml:"enable"`
			Port        int    `yaml:"port"`
			Token       string `yaml:"token"`
			ChannelName string `yaml:"channel_name"`
			TriggerWord string `yaml:"torigger_word"`
		} `yaml:"hook"`
	} `yaml:"slack"`
}

func loadAppConfig() *AppConfig {
	if appConfigFile == "" {
		log.Critical("Missing config file")
		os.Exit(1)
	}
	buf, err := ioutil.ReadFile(appConfigFile)
	if err != nil {
		log.Critical(err)
		os.Exit(1)
	}
	cfg := AppConfig{}
	err = yaml.Unmarshal(buf, &cfg)
	if err != nil {
		log.Critical(err)
		os.Exit(1)
	}
	return &cfg
}

func initLogger() error {
	if logXmlFile == "" {
		l, _ := log.LoggerFromConfigAsString(`<seelog type="sync"><outputs formatid="std:debug"><console/></outputs></seelog>`)
		return log.ReplaceLogger(l)
	}
	l, err := log.LoggerFromConfigAsFile(logXmlFile)
	if err != nil {
		return err
	}
	return log.ReplaceLogger(l)
}
