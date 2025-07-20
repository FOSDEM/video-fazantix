package nullsink

import (
	"github.com/fosdem/fazantix/lib/config"
	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/layer"
)

type NullSink struct {
	frames   layer.FrameForwarder
}

func New(name string, cfg *config.NullSinkCfg, frameCfg *encdec.FrameCfg, alloc encdec.FrameAllocator) *NullSink {
	f := &NullSink{}
	f.frames.Init(
		name,
		&encdec.FrameInfo{
			FrameType: encdec.RGBFrames,
			FrameCfg:  *frameCfg,
		},
		alloc,
	)
	return f
}

func (f *NullSink) Start() bool {
	return true
}

func (f *NullSink) Frames() *layer.FrameForwarder {
	return &f.frames
}

func (f *NullSink) log(msg string, args ...interface{}) {
	f.Frames().Log(msg, args...)
}
