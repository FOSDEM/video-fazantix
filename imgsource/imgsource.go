package imgsource

import (
	"log"
	"os"

	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"

	"github.com/fosdem/vidmix/layer"
	"github.com/go-gl/gl/v2.1/gl"
	"github.com/go-gl/mathgl/mgl32"
)

type ImgSource struct {
	framesRGB chan *image.NRGBA
	path      string
	texture   uint32
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

func (s *ImgSource) Texture(idx int) uint32 {
	return s.texture
}

func (s *ImgSource) FrameType() layer.FrameType {
	return layer.RGBFrames
}

func (s *ImgSource) GenRGBFrames() <-chan *image.NRGBA {
	return s.framesRGB
}

func (s *ImgSource) GenYUVFrames() <-chan *image.YCbCr {
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

	// s.texture = s.setupRGBTexture(s.Width(), s.Height(), s.rgba.Pix)
	// s.isReady = true
	return true
}

func (s *ImgSource) setupRGBTexture(width int, height int, texture []byte) uint32 {
	var id uint32
	gl.GenTextures(1, &id)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, id)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

	borderColor := mgl32.Vec4{0, 0, 0, 0}
	gl.TexParameterfv(gl.TEXTURE_2D, gl.TEXTURE_BORDER_COLOR, &borderColor[0])
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_BORDER)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_BORDER)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(width),
		int32(height),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(texture),
	)
	return id
}
