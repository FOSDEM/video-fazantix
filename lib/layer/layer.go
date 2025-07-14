package layer

import (
	"fmt"
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
	Top     float32
	Left    float32
	Bottom  float32
	Right   float32
	Cx      float32
	Cy      float32
	Opacity float32
}

func normalize(value float32, aspect float32) float32 {
	if value >= 0 {
		return value
	}
	value *= -1
	value *= aspect
	return value
}

func (s *LayerState) Validate() error {
	if s.X != 0 && (s.Left != 0 || s.Right != 0) {
		return fmt.Errorf("cannot set both X and Left or Right for the position")
	}
	if s.Y != 0 && (s.Top != 0 || s.Bottom != 0) {
		return fmt.Errorf("cannot set both Y and Top or Bottom for the position")
	}
	if s.Top != 0 && s.Bottom != 0 && s.Left != 0 && s.Right != 0 {
		return fmt.Errorf("cannot define all four edges for position")
	}

	s.Left = normalize(s.Left, 9.0/16)
	s.Right = normalize(s.Right, 9.0/16)
	s.Top = normalize(s.Top, 1)
	s.Bottom = normalize(s.Bottom, 1)

	if s.Scale == 0 {
		if s.Left != 0 && s.Right != 0 {
			s.Scale = 1.0 - s.Left - s.Right
		} else if s.Top != 0 && s.Bottom != 0 {
			s.Scale = 1.0 - s.Top - s.Bottom
		}
	}

	if s.X == 0 && s.Y == 0 {
		if s.Left != 0 {
			s.X = s.Left
		} else {
			s.X = (1.0 - s.Right) - s.Scale
		}
		if s.Top != 0 {
			s.Y = s.Top
		} else {
			s.Y = (1.0 - s.Bottom) - s.Scale
		}
	}

	if s.Cx != 0 {
		if s.Scale == 0 {
			// Figure out scale from an edge constraint
			if s.Left != 0 {
				s.Scale = (s.Cx - s.Left) * 2
			} else if s.Right != 0 {
				s.Scale = ((1.0 - s.Cx) - s.Right) * 2
			} else {
				return fmt.Errorf("horisontal scale undercontrained")
			}
		}
		s.X = s.Cx - (s.Scale / 2)
	}
	if s.Cy != 0 {
		if s.Scale == 0 {
			// Figure out scale from an edge constraint
			if s.Top != 0 {
				s.Scale = (s.Cy - s.Top) * 2
			} else if s.Bottom != 0 {
				s.Scale = ((1.0 - s.Cy) - s.Bottom) * 2
			} else {
				return fmt.Errorf("vertical scale undercontrained")
			}
		}
		s.Y = s.Cy - (s.Scale / 2)
	}

	return nil
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
