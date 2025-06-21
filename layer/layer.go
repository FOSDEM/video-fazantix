package layer

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
)

type Coordinate struct {
	X float32
	Y float32
}

type Mask struct {
	top    float32
	bottom float32
	left   float32
	right  float32
}

type Layer struct {
	Name string

	IsVisible bool

	Size     Coordinate
	Position Coordinate
	Mask     Mask

	OutputWidth  int
	OutputHeight int
	Squeeze      float32

	Source Source
}

type FrameType int

const (
	YUVFrames FrameType = iota
	RGBFrames
)

type Source interface {
	FrameType() FrameType
	GenRGBFrames() <-chan *image.NRGBA
	GenYUVFrames() <-chan *image.YCbCr
	IsReady() bool
	IsStill() bool
	Width() int
	Height() int
	Start() bool

	Texture(int) uint32
}

func New(name string, src Source, width int, height int) *Layer {
	s := &Layer{Name: name, IsVisible: false}
	s.Size = Coordinate{X: 1.0, Y: 1.0}
	s.Source = src
	s.Squeeze = 1.0
	s.OutputWidth = width
	s.OutputHeight = height
	s.Position = Coordinate{X: 0.5, Y: 0.5}
	s.Mask = Mask{top: 0, bottom: 0, left: 0, right: 0}
	s.Squeeze = (float32(width) / float32(height)) / (float32(s.Source.Width()) / float32(s.Source.Height()))
	return s
}

func (s *Layer) Move(x float32, y float32, size float32) {
	s.Position.X = x
	s.Position.Y = y
	s.Size.X = size
	s.Size.Y = size / s.Squeeze
}
