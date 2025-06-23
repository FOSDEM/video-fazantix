package config

import (
	"fmt"
	"os"

	"github.com/fosdem/fazantix/layer"
	yaml "github.com/goccy/go-yaml"
)

type Config struct {
	Sources map[string]*SourceCfg
	Sinks   map[string]*SinkCfg
	Scenes  map[string]map[string]*layer.LayerState
	Window  *WindowCfg
	Api     *ApiCfg
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
	Z    float32
}

type SourceCfg struct {
	SourceCfgStub
	Cfg interface{}
}

type SinkCfgStub struct {
	Type string
}

type SinkCfg struct {
	SinkCfgStub
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
type FFmpegSinkCfg struct {
	W   int
	H   int
	Cmd string
}

type ImgSourceCfg struct {
	Path string
}

type V4LSourceCfg struct {
	Path string
	Fmt  string
	W    int
	H    int
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
	case "image":
		cfg := ImgSourceCfg{}
		s.Cfg = &cfg
		return yaml.Unmarshal(b, &cfg)
	case "v4l":
		cfg := V4LSourceCfg{}
		s.Cfg = &cfg
		return yaml.Unmarshal(b, &cfg)
	default:
		return fmt.Errorf("unknown source type: %s", s.Type)
	}
}

func (s *SinkCfg) UnmarshalYAML(b []byte) error {
	err := yaml.Unmarshal(b, &s.SinkCfgStub)
	if err != nil {
		return err
	}

	switch s.Type {
	case "ffmpeg_stdin":
		cfg := FFmpegSinkCfg{}
		s.Cfg = &cfg
		return yaml.Unmarshal(b, &cfg)
	default:
		return fmt.Errorf("unknown source type: %s", s.Type)
	}
}

type ApiCfg struct {
	Bind           string
	EnableProfiler bool `yaml:"enable_profiler"`
}
