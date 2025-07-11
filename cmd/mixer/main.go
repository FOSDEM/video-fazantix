package main

import (
	"log"
	"os"

	"github.com/fosdem/fazantix/lib/config"
	"github.com/fosdem/fazantix/lib/mixer"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <config file>", os.Args[0])
	}
	cfg, err := config.Parse(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	mixer.MakeWindowAndMix(cfg)
}
