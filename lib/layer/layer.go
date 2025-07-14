package layer

import (
	"math"
)

type Coordinate struct {
	X float32
	Y float32
}

type Mask struct {
	top    float32
	bottom float32
	left   float32
	right  float32
}

type Layer struct {
	Size     Coordinate
	Position Coordinate
	Mask     Mask

	OutputWidth  int
	OutputHeight int
	Squeeze      float32

	Opacity float32

	Source Source

	targetState *LayerState
}

type LayerState struct {
	X       float32
	Y       float32
	Scale   float32
	Opacity float32
}

func New(src Source, width int, height int) *Layer {
	s := &Layer{}
	s.Size = Coordinate{X: 1.0, Y: 1.0}
	s.Source = src
	s.Squeeze = 1.0
	s.OutputWidth = width
	s.OutputHeight = height
	s.Position = Coordinate{X: 0.5, Y: 0.5}
	s.Mask = Mask{top: 0, bottom: 0, left: 0, right: 0}
	s.Squeeze = (float32(width) / float32(height)) / (float32(s.Source.Frames().Width) / float32(s.Source.Frames().Height))
	if s.Squeeze != s.Squeeze {
		s.Squeeze = 1.0
	}
	return s
}

func (s *Layer) Name() string {
	return s.Source.Frames().Name
}

func (s *Layer) ApplyState(state *LayerState) {
	if state == nil {
		state = &LayerState{}
		if s.targetState != nil {
			*state = *s.targetState
		}
		state.Opacity = 0
	}

	if s.targetState == nil {
		s.Position.X = state.X
		s.Position.Y = state.Y
		s.Size.X = state.Scale
		s.Size.Y = state.Scale / s.Squeeze
		s.Opacity = state.Opacity
	}
	s.targetState = state
}

func (s *Layer) Animate(delta float32) {
	if s.targetState == nil {
		return
	}
	speed := float32(7)
	s.Squeeze = (float32(s.OutputWidth) / float32(s.OutputHeight)) / (float32(s.Source.Frames().Width) / float32(s.Source.Frames().Height))
	s.Position.X = ramp(s.Position.X, s.targetState.X, delta, speed)
	s.Position.Y = ramp(s.Position.Y, s.targetState.Y, delta, speed)
	s.Size.X = ramp(s.Size.X, s.targetState.Scale, delta, speed)
	s.Size.Y = ramp(s.Size.Y, s.targetState.Scale/s.Squeeze, delta, speed)
	s.Opacity = ramp(s.Opacity, s.targetState.Opacity, delta, speed)
}

func (s *Layer) Frames() *FrameForwarder {
	return s.Source.Frames()
}

func ramp(x float32, tgt float32, delta float32, speed float32) float32 {
	return x + (tgt-x)*(1-float32(math.Exp(float64(-speed*delta))))
}
