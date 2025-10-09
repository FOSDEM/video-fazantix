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

	Source    Source
	SourceIdx uint32

	targetTransform *LayerTransform
}

type LayerState struct {
	LayerTransform
	Warp *LayerTransform
}

type LayerTransform struct {
	X       float32
	Y       float32
	Scale   float32
	Opacity float32
}

func New(idx uint32, src Source, width int, height int) *Layer {
	s := &Layer{}
	s.Size = Coordinate{X: 1.0, Y: 1.0}
	s.Source = src
	s.SourceIdx = idx
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

func (s *Layer) ApplyState(state *LayerState, transition bool) {
	var transform LayerTransform
	if state == nil {
		if s.targetTransform != nil {
			transform = *s.targetTransform
		}
		transform.Opacity = 0
	} else {
		transform = state.LayerTransform
	}

	if !transition {
		if state != nil {
			s.Position.X = state.X
			s.Position.Y = state.Y
			s.Size.X = state.Scale
			s.Size.Y = state.Scale / s.Squeeze
			s.Opacity = state.Opacity
		} else {
			s.Opacity = 0.0
		}
	}

	if s.Opacity < (1.0/256.0) && state != nil && state.Warp != nil {
		s.Position.X = state.Warp.X
		s.Position.Y = state.Warp.Y
		s.Size.X = state.Warp.Scale
		s.Size.Y = state.Warp.Scale / s.Squeeze
		s.Opacity = state.Warp.Opacity
	}

	if s.targetTransform == nil {
		base := &transform
		if state != nil && state.Warp != nil {
			base = state.Warp
		}
		s.Position.X = base.X
		s.Position.Y = base.Y
		s.Size.X = base.Scale
		s.Size.Y = base.Scale / s.Squeeze
		s.Opacity = base.Opacity
	}
	s.targetTransform = &transform
}

func (s *Layer) Animate(delta float32, speed float32) {
	if s.targetTransform == nil {
		return
	}
	s.Squeeze = (float32(s.OutputWidth) / float32(s.OutputHeight)) / (float32(s.Source.Frames().Width) / float32(s.Source.Frames().Height))
	s.Position.X = ramp(s.Position.X, s.targetTransform.X, delta, speed)
	s.Position.Y = ramp(s.Position.Y, s.targetTransform.Y, delta, speed)
	s.Size.X = ramp(s.Size.X, s.targetTransform.Scale, delta, speed)
	s.Size.Y = ramp(s.Size.Y, s.targetTransform.Scale/s.Squeeze, delta, speed)
	s.Opacity = ramp(s.Opacity, s.targetTransform.Opacity, delta, speed)
}

func (s *Layer) Frames() *FrameForwarder {
	return s.Source.Frames()
}

func ramp(x float32, tgt float32, delta float32, speed float32) float32 {
	return x + (tgt-x)*(1-float32(math.Exp(float64(-speed*delta))))
}
