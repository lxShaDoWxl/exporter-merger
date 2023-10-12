package main

import (
	"log"
	"prometheus/exporter-merger/cmd"
)

func main() {
	if err := cmd.NewRootCommand().Execute(); err != nil {
		log.Fatal(err)
	}
}
