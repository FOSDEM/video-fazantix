package layer

import (
	"fmt"
	"time"
)

type Stage struct {
	Layers        []*Layer
	SourceIndices []uint32

	LayersByScene        map[string][]*Layer
	SourceIndicesByScene map[string][]uint32

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

func (s *Stage) ActivateScene(sceneName string) error {
	if layers, ok := s.LayersByScene[sceneName]; ok {
		if sourceIndices, ok := s.SourceIndicesByScene[sceneName]; ok {
			s.Layers = layers
			s.SourceIndices = sourceIndices
		} else {
			panic("wtf")
		}
	} else {
		return fmt.Errorf("no such scene: %s", sceneName)
	}
	return nil
}
