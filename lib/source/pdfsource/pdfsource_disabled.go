//go:build !mupdf

package pdfsource

import (
	_ "image/jpeg"
	_ "image/png"

	"github.com/fosdem/fazantix/lib/config"
	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/layer"
)

func init() {
	config.EnableMupdf = false
}

type PdfSource struct {
	frames layer.FrameForwarder
}

func New(name string, cfg *config.HtmlSourceCfg, alloc encdec.FrameAllocator) *PdfSource {
	s := &PdfSource{}
	return s
}

func (s *PdfSource) Start() bool {
	return false
}

func (s *PdfSource) Frames() *layer.FrameForwarder {
	return &s.frames
}
