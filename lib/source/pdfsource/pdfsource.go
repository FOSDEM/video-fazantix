//go:build mupdf

package pdfsource

import (
	_ "image/jpeg"
	_ "image/png"

	"github.com/fosdem/fazantix/lib/config"
	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/layer"
	"github.com/gen2brain/go-fitz"
)

type PdfSource struct {
	path   string
	loaded bool
	width  int
	height int
	page   int

	frames layer.FrameForwarder
}

func New(name string, cfg *config.PdfSourceCfg, alloc encdec.FrameAllocator) *PdfSource {
	s := &PdfSource{
		path:   string(cfg.Path),
		width:  cfg.Width,
		height: cfg.Height,
		frames: layer.FrameForwarder{},
	}
	s.frames.Name = name
	s.frames.InitLogging()

	s.frames.Init(
		name,
		&encdec.FrameInfo{
			FrameType: encdec.RGBAFrames,
			PixFmt:    []uint8{},
			FrameCfg: encdec.FrameCfg{
				Width:              s.width,
				Height:             s.height,
				NumAllocatedFrames: 2,
			},
		},
		alloc,
	)

	s.loaded = true
	return s
}

func (s *PdfSource) Start() bool {
	if !s.loaded {
		return false
	}
	s.frames.HoldFrame = true

	go s.Render()
	return true
}

func (s *PdfSource) Frames() *layer.FrameForwarder {
	return &s.frames
}

func (s *PdfSource) Render() error {
	s.Frames().Log("Rendering PDF")

	pdf, err := fitz.New(s.path)
	if err != nil {
		return err
	}

	s.Frames().Log("PDF has %d pages", pdf.NumPage())

	// Get render size at 72 DPI
	bound, err := pdf.Bound(s.page)
	if err != nil {
		return err
	}

	wdpi := 72.0 / float64(bound.Max.X) * float64(s.width)
	hdpi := 72.0 / float64(bound.Max.Y) * float64(s.height)
	dpi := min(wdpi, hdpi)
	s.Frames().Log("Bounds: %v", bound)

	img, err := pdf.ImageDPI(s.page, dpi)

	frame := s.frames.GetFrameForWriting()
	err = encdec.FrameFromImage(img, frame)
	if err != nil {
		s.Frames().Error("Decode error: %s", err)
		s.frames.FailedWriting(frame)
		return err
	}
	s.frames.FinishedWriting(frame)

	return nil
}

func (s *PdfSource) log(msg string, args ...interface{}) {
	if s.Frames() != nil {
		s.Frames().Log(msg, args...)
	}
}
