package windowsink

import (
	"github.com/fosdem/fazantix/config"
	"github.com/fosdem/fazantix/encdec"
	"github.com/fosdem/fazantix/layer"
)

type WindowSink struct {
	frames layer.FrameForwarder
}

func New(name string, cfg *config.WindowSinkCfg) *WindowSink {
	w := &WindowSink{}
	w.frames.Init(name, encdec.YUV422Frames, []uint8{}, cfg.W, cfg.H)
	return w
}

func (w *WindowSink) Frames() *layer.FrameForwarder {
	return &w.frames
}
