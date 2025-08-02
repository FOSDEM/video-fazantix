package config

import (
	"fmt"

	"github.com/fosdem/fazantix/lib/layer"
)

type LayerCfgExtendedPositioning struct {
	Top    float32
	Left   float32
	Bottom float32
	Right  float32
	Cx     float32
	Cy     float32
}

func applyExtendedPositions(l *layer.LayerTransform, ext *LayerCfgExtendedPositioning) error {
	if l.X != 0 && (ext.Left != 0 || ext.Right != 0) {
		return fmt.Errorf("cannot set both X and Left or Right for the position")
	}
	if l.Y != 0 && (ext.Top != 0 || ext.Bottom != 0) {
		return fmt.Errorf("cannot set both Y and Top or Bottom for the position")
	}
	if ext.Top != 0 && ext.Bottom != 0 && ext.Left != 0 && ext.Right != 0 {
		return fmt.Errorf("cannot define all four edges for position")
	}

	ext.Left = normalize(ext.Left, 9.0/16)
	ext.Right = normalize(ext.Right, 9.0/16)
	ext.Top = normalize(ext.Top, 1)
	ext.Bottom = normalize(ext.Bottom, 1)

	if l.Scale == 0 {
		if ext.Left != 0 && ext.Right != 0 {
			l.Scale = 1.0 - ext.Left - ext.Right
		} else if ext.Top != 0 && ext.Bottom != 0 {
			l.Scale = 1.0 - ext.Top - ext.Bottom
		}
	}

	if l.X == 0 && l.Y == 0 {
		if ext.Left != 0 {
			l.X = ext.Left
		} else {
			l.X = (1.0 - ext.Right) - l.Scale
		}
		if ext.Top != 0 {
			l.Y = ext.Top
		} else {
			l.Y = (1.0 - ext.Bottom) - l.Scale
		}
	}

	if ext.Cx != 0 {
		if l.Scale == 0 {
			// Figure out scale from an edge constraint
			if ext.Left != 0 {
				l.Scale = (ext.Cx - ext.Left) * 2
			} else if ext.Right != 0 {
				l.Scale = ((1.0 - ext.Cx) - ext.Right) * 2
			} else {
				return fmt.Errorf("horisontal scale underconstrained")
			}
		}
		l.X = ext.Cx - (l.Scale / 2)
	}
	if ext.Cy != 0 {
		if l.Scale == 0 {
			// Figure out scale from an edge constraint
			if ext.Top != 0 {
				l.Scale = (ext.Cy - ext.Top) * 2
			} else if ext.Bottom != 0 {
				l.Scale = ((1.0 - ext.Cy) - ext.Bottom) * 2
			} else {
				return fmt.Errorf("vertical scale underconstrained")
			}
		}
		l.Y = ext.Cy - (l.Scale / 2)
	}

	return nil
}
