package imgsource

import (
	"os"

	"image"
	_ "image/jpeg"
	_ "image/png"

	"github.com/fosdem/fazantix/config"
	"github.com/fosdem/fazantix/encdec"
	"github.com/fosdem/fazantix/layer"
)

type ImgSource struct {
	path   string
	loaded bool
	rgba   *image.NRGBA
	img    image.Image

	frames layer.FrameForwarder
}

func New(name string, cfg *config.ImgSourceCfg) *ImgSource {
	s := &ImgSource{}

	s.path = cfg.Path
	s.log("Loading")
	imgFile, err := os.Open(s.path)
	if err != nil {
		s.log("Error opening %s: %s", s.path, err)
		return s
	}

	s.img, _, err = image.Decode(imgFile)
	if err != nil {
		s.log("Error decoding %s: %s", s.path, err)
		return s
	}

	s.rgba = image.NewNRGBA(s.img.Bounds())

	s.frames.Init(
		name,
		encdec.RGBAFrames, s.rgba.Pix,
		s.rgba.Rect.Size().X, s.rgba.Rect.Size().Y,
	)

	if s.rgba.Stride != s.frames.Width*4 {
		s.log("Unsupported stride")
		return s
	}

	s.loaded = true
	return s
}

func (s *ImgSource) Start() bool {
	if !s.loaded {
		return false
	}

	w := s.img.Bounds().Dx()
	h := s.img.Bounds().Dy()
	s.log("Size: %dx%d", w, h)

	s.frames.IsReady = true
	s.frames.IsStill = true

	frame := encdec.NewFrame(encdec.RGBAFrames, w, h)
	err := encdec.FrameFromImage(s.img, frame)
	if err != nil {
		s.log("Decode error: %s", err)
		return false
	}
	s.frames.SendFrame(frame)
	return true
}

func (s *ImgSource) Frames() *layer.FrameForwarder {
	return &s.frames
}

func (s *ImgSource) log(msg string, args ...interface{}) {
	s.Frames().Log(msg, args...)
}
