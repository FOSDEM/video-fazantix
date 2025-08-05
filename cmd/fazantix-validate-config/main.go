package main

import (
	"fmt"
	"log"
	"os"

	"github.com/fosdem/fazantix/lib/config"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <config file>", os.Args[0])
	}
	cfg, err := config.Parse(os.Args[1])
	if err != nil {
		fmt.Printf("Config invalid: %s\n", err)
		os.Exit(1)
	}

	fmt.Print("Config valid!\n\n")

	fmt.Print(cfg)

}
