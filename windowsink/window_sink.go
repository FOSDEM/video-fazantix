package windowsink

import (
	"github.com/fosdem/fazantix/config"
	"github.com/fosdem/fazantix/encdec"
	"github.com/fosdem/fazantix/layer"
)

type WindowSink struct {
	frames layer.FrameForwarder
}

func New(name string, cfg *config.WindowSinkCfg, alloc encdec.FrameAllocator) *WindowSink {
	w := &WindowSink{}
	w.frames.Init(
		name,
		&encdec.FrameInfo{
			FrameType: encdec.RGBFrames,
			PixFmt:    []uint8{},
			FrameCfg:  cfg.FrameCfg,
		},
		alloc,
	)
	return w
}

func (w *WindowSink) Start() bool {
	return true
}

func (w *WindowSink) Frames() *layer.FrameForwarder {
	return &w.frames
}
