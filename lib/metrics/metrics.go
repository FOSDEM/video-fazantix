package metrics

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	FramesForwarded = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "fazantix_source_frames_forwarded_total",
		Help: "Total number of frames forwarded from a source",
	}, []string{"name"})
)

type SourceMetrics struct {
	FramesForwarded prometheus.Counter
}

func NewSourceMetrics(name string) SourceMetrics {
	s := SourceMetrics{
		FramesForwarded: FramesForwarded.WithLabelValues(name),
	}
	s.FramesForwarded.Add(0)
	return s
}

func Serve() error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	server := &http.Server{
		Addr:    ":9099",
		Handler: mux,
	}
	return server.ListenAndServe()
}

func ServeInBackground() {
	log.Println("starting metrics server")
	go func() {
		err := Serve()
		if err != nil {
			log.Fatalf("could not start metrics server: %s", err)
		}
	}()
}
