//go:build mupdf

package pdfsource

import (
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"

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

	doc *fitz.Document

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

	go s.SetDocumenFromPath(s.path)
	return true
}

func (s *PdfSource) Frames() *layer.FrameForwarder {
	return &s.frames
}

func (s *PdfSource) SetDocument(data io.ReadCloser) error {
	doc, err := fitz.NewFromReader(data)
	if err != nil {
		return err
	}
	if s.doc != nil {
		s.doc.Close()
	}
	s.page = 0
	s.doc = doc

	go s.Render()

	return nil
}

func (s *PdfSource) SetDocumenFromPath(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	err = s.SetDocument(file)
	if err != nil {
		return err
	}
	return nil
}

func (s *PdfSource) Render() error {
	s.Frames().Log("Rendering PDF")

	s.Frames().Log("PDF has %d pages", s.doc.NumPage())

	// Get render size at 72 DPI
	bound, err := s.doc.Bound(s.page)
	if err != nil {
		return err
	}

	// Calculate the render DPI required to match the source resolution
	wdpi := 72.0 / float64(bound.Max.X) * float64(s.width)
	hdpi := 72.0 / float64(bound.Max.Y) * float64(s.height)
	dpi := min(wdpi, hdpi)

	img, err := s.doc.ImageDPI(s.page, dpi)

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

func (s *PdfSource) SetPage(page int, relative bool) error {
	newPage := page
	if relative {
		newPage = s.page + page
	}
	if newPage < 0 || newPage > s.doc.NumPage()-1 {
		return fmt.Errorf("slide number out of range")
	}
	s.page = newPage
	go s.Render()
	return nil
}
