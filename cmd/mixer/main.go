package main

import "github.com/fosdem/fazantix/mixer"
import _ "net/http/pprof"
import "log"
import "net/http"

func profiler() {
	log.Println("[pprof] Profiler on :6060")
	log.Println(http.ListenAndServe("localhost:6060", nil))
}

func main() {
	go profiler()
	mixer.MakeWindowAndMix()
}
