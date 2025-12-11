//go:build !mupdf

package pdfsource

import (
	_ "image/jpeg"
	_ "image/png"
	"io"

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

func New(name string, cfg *config.PdfSourceCfg, alloc encdec.FrameAllocator) *PdfSource {
	s := &PdfSource{}
	return s
}

func (s *PdfSource) Start() bool {
	return false
}

func (s *PdfSource) Frames() *layer.FrameForwarder {
	return &s.frames
}

func (s *PdfSource) SetDocument(data io.ReadCloser) error { return nil }

func (s *PdfSource) SetPage(page int, relative bool) error { return nil }
