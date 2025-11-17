//go:build plutobook

package htmlsource

import (
	_ "image/jpeg"
	_ "image/png"

	"github.com/fosdem/fazantix/lib/config"
	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/layer"
	"github.com/fosdem/fazantix/lib/source/htmlsource/plutobook"
)

type HtmlSource struct {
	html   string
	url    string
	loaded bool
	width  int
	height int
	css    string

	frames layer.FrameForwarder
}

func New(name string, cfg *config.HtmlSourceCfg, alloc encdec.FrameAllocator) *HtmlSource {
	s := &HtmlSource{
		html:   cfg.Html,
		url:    cfg.Url,
		width:  cfg.Width,
		height: cfg.Height,
		css:    cfg.Css,
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

func (s *HtmlSource) Start() bool {
	if !s.loaded {
		return false
	}

	go s.Render()
	return true
}

func (s *HtmlSource) Frames() *layer.FrameForwarder {
	return &s.frames
}

func (s *HtmlSource) Render() error {
	s.Frames().Log("Starting render of html source...")
	book := plutobook.New(&plutobook.PageSize{
		Width:  float32(float32(s.width) * plutobook.Pixels),
		Height: float32(float32(s.height) * plutobook.Pixels),
	}, &plutobook.Margins{}, plutobook.MediaTypeScreen)
	canvas := plutobook.NewCanvas(s.width, s.height, plutobook.ImageFormatARGB32)

	var err error
	if s.url != "" {
		s.Frames().Log("Rendering url %s", s.url)
		err = book.LoadUrl(s.url, s.css, "")
	} else {
		s.Frames().Log("Rendering document %s", s.html)
		err = book.LoadHtml(s.html, s.css, "", "/")
	}
	if err != nil {
		return err
	}
	book.RenderDocumentRect(canvas, 0, 0, float32(s.width), float32(s.height))

	data := canvas.GetData()
	s.Frames().Log("Pushing frame, %d bytes", len(data))
	frame := s.frames.GetFrameForWriting()
	if frame == nil {
		panic("got a framedrop while rendering html, not implemented")
	}
	frame.Clear()
	frame.MakeTexture(len(data), s.width, s.height)
	copy(frame.Data, data)
	s.frames.IsReady = true
	s.frames.HoldFrame = true
	s.frames.FinishedWriting(frame)

	return nil
}

func (s *HtmlSource) log(msg string, args ...interface{}) {
	if s.Frames() != nil {
		s.Frames().Log(msg, args...)
	}
}
