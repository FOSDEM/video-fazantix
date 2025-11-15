//go:build !omt

package omtsink

import (
	"github.com/fosdem/fazantix/lib/config"
	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/layer"
)

func init() {
	config.EnableOmt = false
}

type OmtSink struct {
	frames layer.FrameForwarder
}

func New(name string, cfg *config.OmtSinkCfg, frameCfg *encdec.FrameCfg, alloc encdec.FrameAllocator) *OmtSink {
	return &OmtSink{}
}

func (f *OmtSink) Start() bool {
	return false
}

func (f *OmtSink) Frames() *layer.FrameForwarder {
	return &f.frames
}

func (f *OmtSink) SetRate(rate int) {}
