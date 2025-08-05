package config

import (
	"fmt"

	"github.com/fosdem/fazantix/lib/layer"
	yaml "github.com/goccy/go-yaml"
)

type LayerStateCfg struct {
	layer.LayerTransform
	LayerCfgExtendedPositioning
}

type LayerCfg struct {
	LayerStateCfg
	LayerCfgStub
}

type LayerCfgStub struct {
	Warp *LayerStateCfg `yaml:"warp"`
}

func (l *LayerCfg) Validate() error {
	err := l.LayerStateCfg.Validate()
	if err != nil {
		return err
	}

	if l.Warp != nil {
		err = l.Warp.Validate()
		if err != nil {
			return fmt.Errorf("warp config is invalid: %w", err)
		}
	}
	return err
}

func (l *LayerStateCfg) Validate() error {
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
		LayerTransform: l.LayerTransform,
		Warp:           warp,
	}
}

func (l *LayerStateCfg) UnmarshalYAML(b []byte) error {
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

func (l *LayerCfg) UnmarshalYAML(b []byte) error {
	err := yaml.Unmarshal(b, &l.LayerCfgStub)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(b, &l.LayerStateCfg)
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
