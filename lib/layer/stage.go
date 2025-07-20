package layer

type Stage struct {
	Layers       []*Layer
	HFlip        bool
	VFlip        bool
	Sink         Sink
	DefaultScene string
	PreviewFor   string
}

type Sink interface {
	Frames() *FrameForwarder
	Start() bool
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
