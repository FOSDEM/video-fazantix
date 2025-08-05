package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	FramesForwarded = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "fazantix_stream_frames_forwarded_total",
		Help: "Total number of frames forwarded as part of stream",
	}, []string{"name"})
	FramesDropped = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "fazantix_stream_frames_dropped_total",
		Help: "Total number of frames dropped as part of stream",
	}, []string{"name"})
)

type StreamMetrics struct {
	FramesForwarded prometheus.Counter
	FramesDropped   prometheus.Counter
}

func NewStreamMetrics(name string) StreamMetrics {
	s := StreamMetrics{
		FramesForwarded: FramesForwarded.WithLabelValues(name),
		FramesDropped:   FramesDropped.WithLabelValues(name),
	}
	s.FramesForwarded.Add(0)
	s.FramesDropped.Add(0)
	return s
}

// Handler should usually be mounted at /metrics
func Handler() http.Handler {
	return promhttp.Handler()
}
