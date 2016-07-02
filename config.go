package main

import (
	"flag"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
)

var (
	appConfigFile string
)

func init() {
	flag.StringVar(&appConfigFile, "cfg", "", "config yaml file")
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
		log.Fatal("[FATAL] Missing config file")
	}
	buf, err := ioutil.ReadFile(appConfigFile)
	if err != nil {
		log.Fatal("[FATAL] ", err)
	}
	cfg := AppConfig{}
	err = yaml.Unmarshal(buf, &cfg)
	if err != nil {
		log.Fatal("[FATAL] ", err)
	}
	return &cfg
}
