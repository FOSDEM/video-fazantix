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
	framesRGB chan *image.NRGBA
	path      string
	isReady   bool
	loaded    bool
	rgba      *image.RGBA
	img       image.Image
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
	s.loaded = true
	return s
}

func (s *ImgSource) FrameType() layer.FrameType {
	return layer.RGBFrames
}

func (s *ImgSource) GenRGBFrames() <-chan *image.NRGBA {
	return s.framesRGB
}

func (s *ImgSource) PixFmt() []uint8 {
	return s.rgba.Pix
}

func (s *ImgSource) GenYUV422Frames() <-chan *image.YCbCr {
	panic("I have no yuv frames")
}

func (s *ImgSource) Width() int {
	return s.rgba.Rect.Size().X
}

func (s *ImgSource) Height() int {
	return s.rgba.Rect.Size().Y
}

func (s *ImgSource) IsReady() bool {
	return s.isReady
}

func (s *ImgSource) IsStill() bool {
	return true
}

func (s *ImgSource) Start() bool {
	if !s.loaded {
		return false
	}

	draw.Draw(s.rgba, s.rgba.Bounds(), s.img, image.Point{0, 0}, draw.Src)
	log.Printf("[%s] Size: %dx%d", s.path, s.rgba.Bounds().Dx(), s.rgba.Bounds().Dy())

	s.isReady = true
	return true
}
