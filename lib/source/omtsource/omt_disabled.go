//go:build !omt

package omtsource

import (
	"github.com/fosdem/fazantix/lib/config"
	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/layer"
)

type OmtSource struct {
	frames layer.FrameForwarder
}

func New(name string, cfg *config.OmtSourceCfg, alloc encdec.FrameAllocator) *OmtSource {
	return &OmtSource{}
}

func (f *OmtSource) Start() bool {
	return false
}

func (f *OmtSource) Frames() *layer.FrameForwarder {
	return &f.frames
}
