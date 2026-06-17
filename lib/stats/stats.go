package stats

import (
	"fmt"
	"time"

	"github.com/fosdem/fazantix/lib/rendering"
)

type Stats struct {
	TextureUpload      uint64  `json:"texture_upload" example:"211507200"`
	TextureUploadAvgGb float64 `json:"texture_upload_avg_gb" example:"0.03411996282883119"`
	Uptime             float64 `json:"uptime" example:"22.355897797"`
	FPS                uint64  `json:"fps" example:"60"`
	WsClients          int     `json:"ws_clients" example:"1"`
	FPSAvg             float64 `json:"fps_avg" example:"60.0"`

	frameCounter uint64
	frameTimer   time.Time
	start        time.Time
	fpsHistory   [9]uint64
	fpsIndex     int
}

func New() *Stats {
	s := &Stats{}
	s.start = time.Now()
	return s
}

func (s *Stats) Update() {
	s.frameCounter++
	if time.Since(s.frameTimer) > 1*time.Second {
		s.FPS = s.frameCounter
		s.frameCounter = 0
		s.fpsHistory[s.fpsIndex] = s.FPS
		s.fpsIndex++
		if s.fpsIndex > len(s.fpsHistory)-1 {
			s.fpsIndex = 0
		}
		sum := uint64(0)
		for _, h := range s.fpsHistory {
			sum += h
		}
		s.FPSAvg = float64(sum) / float64(len(s.fpsHistory))
		s.frameTimer = time.Now()
	}

	s.Uptime = float64(time.Since(s.start).Nanoseconds()) / 1e9
	s.TextureUpload = rendering.TextureUploadCounter
	s.TextureUploadAvgGb = float64(s.TextureUpload) / (s.Uptime * 1024 * 1024 * 1024)
}

func (s *Stats) Print() {
	fmt.Printf("BENCHMARK: fps:%d avg:%f\n", s.FPS, s.FPSAvg)
}
