package main

import (
	"flag"
	log "github.com/cihub/seelog"
)

var (
	logXmlFile string
)

func init() {
	flag.StringVar(&logXmlFile, "log", "", "log xml file")
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
