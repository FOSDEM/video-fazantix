package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/utils"
	yaml "github.com/goccy/go-yaml"
)

type Config struct {
	Sources        map[string]*SourceCfg
	Scenes         map[string]*SceneCfg
	Stages         map[string]*StageCfg `yaml:"sinks"`
	FallbackColour string               `yaml:"fallback_colour"`
	Api            *ApiCfg
}

func Parse(filename string) (*Config, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("could not open %s: %s", filename, err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			_ = fmt.Errorf("could not close %s: %s", filename, err)
		}
	}(f)

	absFilename, err := filepath.Abs(filename)
	if err != nil {
		return nil, fmt.Errorf("somehow, %s is malformed: %w", filename, err)
	}
	UnmarshalBase = filepath.Dir(absFilename)

	m := yaml.NewDecoder(f)
	cfg := &Config{}
	err = m.Decode(cfg)
	if err != nil {
		return nil, err
	}
	err = cfg.Validate()
	if err != nil {
		return nil, err
	}
	return cfg, err
}

func (c *Config) Validate() error {
	var err error
	if len(c.Sources) < 1 {
		return fmt.Errorf("at least one source should be defined")
	}
	if len(c.Stages) < 1 {
		return fmt.Errorf("at least one sink should be defined")
	}
	for k, v := range c.Sources {
		err = v.Validate()
		if err != nil {
			return fmt.Errorf("source %s is invalid: %w", k, err)
		}
		if _, ok := c.Sources[v.Fallback]; !ok && v.Fallback != "" {
			return fmt.Errorf("%s cannot be used as fallback source (no such source)", v.Fallback)
		}
	}
	for k, v := range c.Stages {
		err = v.Validate()
		if err != nil {
			return fmt.Errorf("sink %s is invalid: %w", k, err)
		}
		if _, ok := c.Scenes[v.DefaultScene]; !ok {
			return fmt.Errorf("scene %s, which is %s's default scene, does not exist", v.DefaultScene, k)
		}
	}
	for k, v := range c.Scenes {
		for i, layerCfg := range v.Layers {
			err = layerCfg.Validate()
			if err != nil {
				return fmt.Errorf("scene %s layer %d is invalid: %w", k, i, err)
			}
			if layerCfg.SourceName != "" {
				if _, ok := c.Sources[layerCfg.SourceName]; !ok {
					return fmt.Errorf("scene %s layer %d refers to non-existant source %s", k, i, layerCfg.SourceName)
				}
			}
			if layerCfg.StageName != "" {
				if _, ok := c.Stages[layerCfg.StageName]; !ok {
					return fmt.Errorf("scene %s layer %d refers to non-existant source %s", k, i, layerCfg.SourceName)
				}
			}
		}
	}

	if c.FallbackColour == "" {
		return fmt.Errorf("please set fallback_colour in the config")
	}
	if !utils.ColourValidate(c.FallbackColour) {
		return fmt.Errorf("%s is not a valid RGBA hex colour", c.FallbackColour)
	}
	return nil
}

func (c *Config) String() string {
	var b strings.Builder
	b.WriteString("Sources:\n")

	for k, v := range c.Sources {
		b.WriteString(fmt.Sprintf("  %s (%s)\n", k, v.Type))
	}

	b.WriteString("\nSinks:\n")
	for k, v := range c.Stages {
		b.WriteString(fmt.Sprintf("  %s (%s)\n", k, v.Type))
	}

	b.WriteString("\nScenes:\n")
	for k := range c.Scenes {
		b.WriteString(fmt.Sprintf("  %s\n", k))
	}

	return b.String()
}

type SourceCfgStub struct {
	Type      string
	Z         float32
	MakeScene bool
	Tag       string
	Label     string
	Fallback  string
}

type SceneCfg struct {
	Tag    string
	Label  string
	Layers []*LayerCfg
}

type StageCfgStub struct {
	Type             string
	DefaultScene     string `yaml:"default_scene"`
	PreviewFor       string `yaml:"preview_for"`
	TransitionTimeMs *int   `yaml:"transition_time_ms"`
	encdec.FrameCfg  `yaml:"frames"`
}

type Valid interface {
	Validate() error
}

type StageCfg struct {
	StageCfgStub
	SinkCfg Valid
}

type SourceCfg struct {
	SourceCfgStub
	Cfg Valid
}

type SinkCfgStub struct {
	Type string
}

type FFmpegSourceCfg struct {
	encdec.FrameCfg `yaml:"frames"`
	Cmd             string
}
type FFmpegSinkCfg struct {
	Cmd string
}

type WindowSinkCfg struct {
}

type ImgSourceCfg struct {
	Path    CfgPath
	Width   int
	Height  int
	Inotify bool
}

type V4LSourceCfg struct {
	encdec.FrameCfg    `yaml:"frames"`
	Path               string
	Fmt                string
	NumFramesInWriting int `yaml:"num_frames_in_writing"`
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

func (s *StageCfg) UnmarshalYAML(b []byte) error {
	err := yaml.Unmarshal(b, &s.StageCfgStub)
	if err != nil {
		return err
	}

	switch s.Type {
	case "ffmpeg_stdin":
		cfg := FFmpegSinkCfg{}
		s.SinkCfg = &cfg
		return yaml.Unmarshal(b, &cfg)
	case "window":
		cfg := WindowSinkCfg{}
		s.SinkCfg = &cfg
		return yaml.Unmarshal(b, &cfg)
	default:
		return fmt.Errorf("unknown stage sink type: %s", s.Type)
	}
}

type ApiCfg struct {
	Bind           string
	EnableProfiler bool `yaml:"enable_profiler"`
}

func (s *StageCfg) Validate() error {
	if s.DefaultScene == "" {
		return fmt.Errorf("default scene must be specified")
	}

	if s.TransitionTimeMs == nil {
		return fmt.Errorf("transition_time_ms must be specified")
	} else if *s.TransitionTimeMs < 0 {
		return fmt.Errorf("transition_time_ms must be nonnegative")
	}

	isWindow := false
	if _, ok := s.SinkCfg.(*WindowSinkCfg); ok {
		isWindow = true
	}
	err := s.FrameCfg.Validate(isWindow)
	if err != nil {
		return fmt.Errorf("invalid frame config: %w", err)
	}
	return s.SinkCfg.Validate()
}

func (s *SourceCfg) Validate() error {
	return s.Cfg.Validate()
}

func (s *ImgSourceCfg) Validate() error {
	if s.Path == "" {
		if s.Width == 0 && s.Height == 0 {
			return fmt.Errorf("image path or size must be specified")
		}
		if s.Inotify {
			return fmt.Errorf("cannot enable inotify for an imagesource without path")
		}
	} else {
		if s.Width != 0 || s.Height != 0 {
			return fmt.Errorf("image path or size can't both be specified")
		}
	}
	return nil
}

func (s *FFmpegSourceCfg) Validate() error {
	if s.Cmd == "" {
		return fmt.Errorf("ffmpeg cmd must be specified")
	}
	return s.FrameCfg.Validate(false)
}

func (s *FFmpegSinkCfg) Validate() error {
	if s.Cmd == "" {
		return fmt.Errorf("ffmpeg cmd must be specified")
	}
	return nil
}

func (s *WindowSinkCfg) Validate() error {
	return nil
}

func (s *V4LSourceCfg) Validate() error {
	if s.Path == "" {
		return fmt.Errorf("path to video device must be specified")
	}

	err := s.FrameCfg.Validate(false)
	if err != nil {
		return err
	}

	if s.NumFramesInWriting < 2 || s.NumFramesInWriting > s.FrameCfg.NumAllocatedFrames-2 {
		return fmt.Errorf("a v4l source should have at least two frames in writing (%d) (`num_frames_in_writing`) but also at least two allocated frames that are not for writing", s.NumFramesInWriting)
	}
	return nil
}
