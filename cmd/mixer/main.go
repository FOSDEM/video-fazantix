package main

import (
	"log"
	"os"

	"github.com/fosdem/fazantix/config"
	"github.com/fosdem/fazantix/mixer"
)

func main() {
	cfg, err := config.Parse(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	mixer.MakeWindowAndMix(cfg)
}
