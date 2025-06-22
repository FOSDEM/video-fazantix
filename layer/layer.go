package layer

import (
	"github.com/fosdem/fazantix/encdec"
	"github.com/fosdem/fazantix/rendering"
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

	TextureIDs [3]uint32
}

type Source interface {
	Frames() *FrameForwarder
	Start() bool
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
	s.Squeeze = (float32(width) / float32(height)) / (float32(s.Source.Frames().Width) / float32(s.Source.Frames().Height))
	return s
}

func (s *Layer) Move(x float32, y float32, size float32) {
	s.Position.X = x
	s.Position.Y = y
	s.Size.X = size
	s.Size.Y = size / s.Squeeze
}

func (s *Layer) SetupTextures() {
	width := s.Source.Frames().Width
	height := s.Source.Frames().Height

	switch s.Frames().FrameType {
	case encdec.YUV422Frames:
		s.TextureIDs[0] = rendering.SetupYUVTexture(width, height)
		s.TextureIDs[1] = rendering.SetupYUVTexture(width/2, height)
		s.TextureIDs[2] = rendering.SetupYUVTexture(width/2, height)
	case encdec.RGBFrames:
		s.TextureIDs[0] = rendering.SetupRGBTexture(width, height, s.Frames().PixFmt)
	}
}

func (s *Layer) Frames() *FrameForwarder {
	return s.Source.Frames()
}
