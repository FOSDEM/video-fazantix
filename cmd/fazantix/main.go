package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"runtime"

	"github.com/fosdem/fazantix/lib/config"
	fazantixLog "github.com/fosdem/fazantix/lib/log"
	"github.com/fosdem/fazantix/lib/mixer"
)

var programLevel = new(slog.LevelVar) // Info by default
var debugFlag = flag.Bool("debug", false, "Set loglevel to debug")
var benchFlag = flag.Bool("benchmark", false, "Run a benchmark")

func init() {
	// The OpenGL stuff must be in one thread
	runtime.LockOSThread()
}

func main() {
	slog.SetDefault(slog.New(fazantixLog.NewHandler(&slog.HandlerOptions{
		Level: programLevel,
	})))
	flag.Parse()
	if *debugFlag {
		programLevel.Set(slog.LevelDebug)
	}

	if flag.NArg() != 1 {
		slog.Error(fmt.Sprintf("Usage: %s <config file>", os.Args[0]))
		os.Exit(1)
	}
	cfg, err := config.Parse(flag.Arg(0))
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	if *benchFlag {
		slog.Warn("Running in benchmark mode")
	}
	mixer.MakeWindowAndMix(cfg, *benchFlag)
}
