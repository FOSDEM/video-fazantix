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

func (l *LayerCfg) Validate() error {
	if l.X != 0 && (l.Left != 0 || l.Right != 0) {
		return fmt.Errorf("cannot set both X and Left or Right for the position")
	}
	if l.Y != 0 && (l.Top != 0 || l.Bottom != 0) {
		return fmt.Errorf("cannot set both Y and Top or Bottom for the position")
	}
	if l.Top != 0 && l.Bottom != 0 && l.Left != 0 && l.Right != 0 {
		return fmt.Errorf("cannot define all four edges for position")
	}

	l.Left = normalize(l.Left, 9.0/16)
	l.Right = normalize(l.Right, 9.0/16)
	l.Top = normalize(l.Top, 1)
	l.Bottom = normalize(l.Bottom, 1)

	if l.Scale == 0 {
		if l.Left != 0 && l.Right != 0 {
			l.Scale = 1.0 - l.Left - l.Right
		} else if l.Top != 0 && l.Bottom != 0 {
			l.Scale = 1.0 - l.Top - l.Bottom
		}
	}

	if l.X == 0 && l.Y == 0 {
		if l.Left != 0 {
			l.X = l.Left
		} else {
			l.X = (1.0 - l.Right) - l.Scale
		}
		if l.Top != 0 {
			l.Y = l.Top
		} else {
			l.Y = (1.0 - l.Bottom) - l.Scale
		}
	}

	if l.Cx != 0 {
		if l.Scale == 0 {
			// Figure out scale from an edge constraint
			if l.Left != 0 {
				l.Scale = (l.Cx - l.Left) * 2
			} else if l.Right != 0 {
				l.Scale = ((1.0 - l.Cx) - l.Right) * 2
			} else {
				return fmt.Errorf("horisontal scale undercontrained")
			}
		}
		l.X = l.Cx - (l.Scale / 2)
	}
	if l.Cy != 0 {
		if l.Scale == 0 {
			// Figure out scale from an edge constraint
			if l.Top != 0 {
				l.Scale = (l.Cy - l.Top) * 2
			} else if l.Bottom != 0 {
				l.Scale = ((1.0 - l.Cy) - l.Bottom) * 2
			} else {
				return fmt.Errorf("vertical scale undercontrained")
			}
		}
		l.Y = l.Cy - (l.Scale / 2)
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
