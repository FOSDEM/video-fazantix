package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	FramesWritten = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "fazantix_stream_frames_written_total",
		Help: "Total number of frames written as part of stream",
	}, []string{"name"})
	FramesRequested = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "fazantix_stream_frames_requested_total",
		Help: "Total number of frames requested from readers as part of stream",
	}, []string{"name"})
	FramesRead = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "fazantix_stream_frames_read_total",
		Help: "Total number of frames actually read by readers as part of stream",
	}, []string{"name"})
	FramesDropped = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "fazantix_stream_frames_dropped_total",
		Help: "Total number of frames dropped as part of stream",
	}, []string{"name"})
)

type StreamMetrics struct {
	FramesRequested prometheus.Counter
	FramesRead      prometheus.Counter
	FramesWritten   prometheus.Counter
	FramesDropped   prometheus.Counter
}

func NewStreamMetrics(name string) StreamMetrics {
	s := StreamMetrics{
		FramesRequested: FramesRequested.WithLabelValues(name),
		FramesRead:      FramesRead.WithLabelValues(name),
		FramesWritten:   FramesWritten.WithLabelValues(name),
		FramesDropped:   FramesDropped.WithLabelValues(name),
	}
	s.FramesRequested.Add(0)
	s.FramesRead.Add(0)
	s.FramesWritten.Add(0)
	s.FramesDropped.Add(0)
	return s
}

// Handler should usually be mounted at /metrics
func Handler() http.Handler {
	return promhttp.Handler()
}
