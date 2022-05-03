package main

import (
	"github.com/jkreileder/exporter-merger/cmd"
	log "github.com/sirupsen/logrus"
)

func main() {
	if err := cmd.NewRootCommand().Execute(); err != nil {
		log.Fatal(err)
	}
}
