package config

import (
	"fmt"

	"github.com/fosdem/fazantix/lib/layer"
	yaml "github.com/goccy/go-yaml"
)

type LayerTransformCfg struct {
	layer.LayerTransform
	LayerCfgExtendedPositioning
}

type LayerCfg struct {
	SourceName string             `yaml:"source"`
	Transform  *LayerTransformCfg `yaml:"transform"`
	Warp       *LayerTransformCfg `yaml:"warp"`
}

func (l *LayerCfg) Validate() error {
	if l.Transform == nil {
		return fmt.Errorf("please add a 'transform' key to the layer definition")
	}

	if l.SourceName == "" {
		return fmt.Errorf("source must be specified")
	}

	err := l.Transform.Validate()
	if err != nil {
		return fmt.Errorf("invalid layer state definition: %w", err)
	}

	if l.Warp != nil {
		err = l.Warp.Validate()
		if err != nil {
			return fmt.Errorf("warp config is invalid: %w", err)
		}
	}
	return err
}

func (l *LayerTransformCfg) Validate() error {
	return applyExtendedPositions(&l.LayerTransform, &l.LayerCfgExtendedPositioning)
}

func (l *LayerCfg) CopyState() *layer.LayerState {
	if l == nil {
		return nil
	}
	var warp *layer.LayerTransform
	if l.Warp != nil {
		warp = &layer.LayerTransform{}
		*warp = l.Warp.LayerTransform
	}

	return &layer.LayerState{
		LayerTransform: l.Transform.LayerTransform,
		Warp:           warp,
	}
}

func (l *LayerTransformCfg) UnmarshalYAML(b []byte) error {
	err := yaml.Unmarshal(b, &l.LayerTransform)
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
