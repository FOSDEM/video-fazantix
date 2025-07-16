package imgsource

import (
	"os"

	"image"
	_ "image/jpeg"
	_ "image/png"

	"github.com/fosdem/fazantix/lib/config"
	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/layer"
)

type ImgSource struct {
	path   string
	loaded bool
	rgba   *image.NRGBA
	img    image.Image

	frames layer.FrameForwarder
}

func New(name string, cfg *config.ImgSourceCfg, alloc encdec.FrameAllocator) *ImgSource {
	s := &ImgSource{}
	s.frames.Name = name

	s.path = string(cfg.Path)
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
		&encdec.FrameInfo{
			FrameType: encdec.RGBAFrames,
			PixFmt:    s.rgba.Pix,
			FrameCfg: encdec.FrameCfg{
				Width:              s.rgba.Rect.Size().X,
				Height:             s.rgba.Rect.Size().Y,
				NumAllocatedFrames: 2,
			},
		},
		alloc,
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
	s.frames.HoldFrame = layer.Hold
	err := s.SetImage(s.img)
	return err == nil
}

func (s *ImgSource) Frames() *layer.FrameForwarder {
	return &s.frames
}

func (s *ImgSource) log(msg string, args ...interface{}) {
	s.Frames().Log(msg, args...)
}

func (s *ImgSource) GetImage() image.Image {
	return s.img
}

func (s *ImgSource) SetImage(newImage image.Image) error {
	s.img = newImage
	s.rgba = image.NewNRGBA(s.img.Bounds())
	frame := s.frames.GetFrameForWriting()
	err := encdec.FrameFromImage(s.img, frame)
	if err != nil {
		s.log("Decode error: %s", err)
		s.frames.FailedWriting(frame)
		return err
	}
	s.frames.FinishedWriting(frame)
	return nil
}
