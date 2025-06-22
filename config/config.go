package config

import (
	"fmt"
	"os"

	"github.com/fosdem/fazantix/layer"
	yaml "github.com/goccy/go-yaml"
)

type Config struct {
	Sources map[string]*SourceCfg
	Scenes  map[string]map[string]*layer.LayerState
	Window  *WindowCfg
}

func Parse(filename string) (*Config, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("could not open %s: %s", filename, err)
	}
	defer f.Close()

	m := yaml.NewDecoder(f)
	cfg := &Config{}
	err = m.Decode(cfg)
	return cfg, err
}

type SourceCfgStub struct {
	Type string
}

type SourceCfg struct {
	SourceCfgStub
	Cfg interface{}
}

type WindowCfg struct {
	W int
	H int
}

type FFmpegSourceCfg struct {
	W   int
	H   int
	Cmd string
}

func (s *SourceCfg) UnmarshalYAML(b []byte) error {
	err := yaml.Unmarshal(b, &s.SourceCfgStub)
	if err != nil {
		return err
	}

	switch s.Type {
	case "ffmpeg_stdout":
		cfg := FFmpegSourceCfg{}
		s.Cfg = &cfg
		return yaml.Unmarshal(b, &cfg)
	default:
		return fmt.Errorf("unknown source type: %s", s.Type)
	}
}
