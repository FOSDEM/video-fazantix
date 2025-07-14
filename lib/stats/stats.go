package stats

import (
	"time"

	"github.com/fosdem/fazantix/lib/rendering"
)

type Stats struct {
	TextureUpload      uint64  `json:"texture_upload"`
	TextureUploadAvgGb float64 `json:"texture_upload_avg_gb"`
	Uptime             float64 `json:"uptime"`
	FPS                uint64  `json:"fps"`
	WsClients          int     `json:"ws_clients"`

	frameCounter uint64
	frameTimer   time.Time
	start        time.Time
}

func New() *Stats {
	s := &Stats{}
	s.start = time.Now()
	return s
}

func (s *Stats) Update() {
	s.frameCounter++
	if time.Since(s.frameTimer) > 1*time.Second {
		s.frameTimer = time.Now()
		s.FPS = s.frameCounter
		s.frameCounter = 0
		s.frameTimer = time.Now()
	}

	s.Uptime = float64(time.Since(s.start).Nanoseconds()) / 1e9
	s.TextureUpload = rendering.TextureUploadCounter
	s.TextureUploadAvgGb = float64(s.TextureUpload) / (s.Uptime * 1024 * 1024 * 1024)
}
