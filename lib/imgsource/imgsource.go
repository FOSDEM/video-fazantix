package imgsource

import (
	"os"
	"time"

	"image"
	_ "image/jpeg"
	_ "image/png"

	"github.com/fosdem/fazantix/lib/config"
	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/layer"
	"github.com/jhenstridge/go-inotify"
)

type ImgSource struct {
	path    string
	loaded  bool
	rgba    *image.NRGBA
	img     image.Image
	inotify bool

	frames layer.FrameForwarder
}

func New(name string, cfg *config.ImgSourceCfg, alloc encdec.FrameAllocator) *ImgSource {
	s := &ImgSource{}
	s.frames.Name = name
	s.inotify = cfg.Inotify

	err := s.LoadImage(string(cfg.Path))
	if err != nil {
		return nil
	}
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

func (s *ImgSource) watch() {
	watcher, err := inotify.NewWatcher()
	if err != nil {
		return
	}
	defer func(watcher *inotify.Watcher) {
		err := watcher.Close()
		if err != nil {
			return
		}
	}(watcher)

	_, err = watcher.Watch(s.path)
	if err != nil {
		s.log("Could not start inotify watcher: %s", err)
		return
	}

	for ev := range watcher.Event {
		if ev.Mask&inotify.IN_CLOSE_WRITE != 0 {
			s.log("Reloading image due to inotify event")
			time.Sleep(100 * time.Millisecond)

			err := s.LoadImage(s.path)
			if err != nil {
				s.log("Error loading image: %s", err)
				continue
			}
			err = s.SetImage(s.img)
			if err != nil {
				s.log("Error setting image: %s", err)
				continue
			}
		}
	}
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

	if s.inotify {
		go s.watch()
	}

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

func (s *ImgSource) LoadImage(newPath string) error {
	s.path = newPath
	s.log("Loading %s", s.path)
	imgFile, err := os.Open(s.path)
	if err != nil {
		s.log("Error opening: %s", err)
		return err
	}

	s.img, _, err = image.Decode(imgFile)
	if err != nil {
		s.log("Error decoding: %s", err)
		return err
	}

	s.rgba = image.NewNRGBA(s.img.Bounds())
	return nil
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
