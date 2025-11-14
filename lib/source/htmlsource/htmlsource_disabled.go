//go:build !plutobook

package htmlsource

import (
	_ "image/jpeg"
	_ "image/png"

	"github.com/fosdem/fazantix/lib/config"
	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/layer"
)

func init() {
	config.EnablePlutobook = false
}

type HtmlSource struct {
	frames layer.FrameForwarder
}

func New(name string, cfg *config.HtmlSourceCfg, alloc encdec.FrameAllocator) *HtmlSource {
	s := &HtmlSource{}
	return s
}

func (s *HtmlSource) Start() bool {
	return false
}

func (s *HtmlSource) Frames() *layer.FrameForwarder {
	return &s.frames
}
