package imgsource

import (
	"log"
	"os"

	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"

	"github.com/fosdem/vidmix/layer"
)

type ImgSource struct {
	path   string
	loaded bool
	rgba   *image.RGBA
	img    image.Image

	frames layer.FrameForwarder
}

func New(path string) *ImgSource {
	s := &ImgSource{}

	s.path = path

	log.Printf("[%s] Loading", s.path)
	imgFile, err := os.Open(s.path)
	if err != nil {
		log.Printf("[%s] Error: %s", s.path, err)
		return s
	}

	s.img, _, err = image.Decode(imgFile)
	if err != nil {
		log.Printf("[%s] Error: %s", s.path, err)
		return s
	}

	s.rgba = image.NewRGBA(s.img.Bounds())
	if s.rgba.Stride != s.Width()*4 {
		log.Printf("[%s] Unsupported stride", s.path)
		return s
	}

	s.frames.Init()
	s.frames.PixFmt = s.rgba.Pix
	s.frames.FrameType = layer.RGBFrames

	s.loaded = true
	return s
}

func (s *ImgSource) Start() bool {
	if !s.loaded {
		return false
	}

	draw.Draw(s.rgba, s.rgba.Bounds(), s.img, image.Point{0, 0}, draw.Src)
	log.Printf("[%s] Size: %dx%d", s.path, s.rgba.Bounds().Dx(), s.rgba.Bounds().Dy())

	s.frames.IsReady = true
	s.frames.IsStill = true
	return true
}

func (s *ImgSource) Frames() *layer.FrameForwarder {
	return &s.frames
}
