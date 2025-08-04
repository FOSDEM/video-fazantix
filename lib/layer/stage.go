package layer

import "time"

type Stage struct {
	Layers       []*Layer
	HFlip        bool
	VFlip        bool
	Sink         Sink
	DefaultScene string
	PreviewFor   string
	Speed        float32
}

type Sink interface {
	Frames() *FrameForwarder
	Start() bool
}

func (s *Stage) SetSpeed(d time.Duration) {
	s.Speed = float32(7.0 / d.Seconds())
}

func (s *Stage) StageData() uint32 {
	data := uint32(0)
	if s.HFlip {
		data += 1
	}
	if s.VFlip {
		data += 2
	}
	return data
}
