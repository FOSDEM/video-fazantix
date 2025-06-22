package imgsource

import (
	"log"
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

	name string

	frames layer.FrameForwarder
}

func New(name string, cfg *config.ImgSourceCfg) *ImgSource {
	s := &ImgSource{}

	s.path = cfg.Path
	s.name = name

	log.Printf("[%s] Loading", s.name)
	imgFile, err := os.Open(s.path)
	if err != nil {
		log.Printf("[%s] Error: %s", s.name, err)
		return s
	}

	s.img, _, err = image.Decode(imgFile)
	if err != nil {
		log.Printf("[%s] Error: %s", s.name, err)
		return s
	}

	s.rgba = image.NewNRGBA(s.img.Bounds())

	s.frames.Init(
		encdec.RGBFrames, s.rgba.Pix,
		s.rgba.Rect.Size().X, s.rgba.Rect.Size().Y,
	)

	if s.rgba.Stride != s.frames.Width*4 {
		log.Printf("[%s] Unsupported stride", s.name)
		return s
	}

	s.loaded = true
	return s
}

func (i *ImgSource) Name() string {
	return i.name
}

func (s *ImgSource) Start() bool {
	if !s.loaded {
		return false
	}

	w := s.img.Bounds().Dx()
	h := s.img.Bounds().Dy()
	log.Printf("[%s] Size: %dx%d", s.name, w, h)

	s.frames.IsReady = true
	s.frames.IsStill = true

	frame := encdec.NewFrame(encdec.RGBFrames, w, h)
	err := encdec.FrameFromImage(s.img, frame)
	if err != nil {
		log.Printf("[%s] Decode error: %s", s.name, err)
		return false
	}
	s.frames.SendFrame(frame)
	return true
}

func (s *ImgSource) Frames() *layer.FrameForwarder {
	return &s.frames
}
