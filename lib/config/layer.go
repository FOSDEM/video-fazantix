package config

import (
	"fmt"

	"github.com/fosdem/fazantix/lib/layer"
	yaml "github.com/goccy/go-yaml"
)

type LayerCfg struct {
	layer.LayerState
	LayerCfgExtendedPositioning
}

type LayerCfgExtendedPositioning struct {
	Top    float32
	Left   float32
	Bottom float32
	Right  float32
	Cx     float32
	Cy     float32
}

func (s *LayerCfg) Validate() error {
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

func (l *LayerCfg) CopyState() *layer.LayerState {
	if l == nil {
		return nil
	}
	ls := l.LayerState
	return &ls
}

func (l *LayerCfg) UnmarshalYAML(b []byte) error {
	err := yaml.Unmarshal(b, &l.LayerState)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(b, &l.LayerCfgExtendedPositioning)
	if err != nil {
		return err
	}

	return nil
}

func normalize(value float32, aspect float32) float32 {
	if value >= 0 {
		return value
	}
	value *= -1
	value *= aspect
	return value
}
