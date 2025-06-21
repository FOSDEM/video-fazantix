package layer

import (
	"image"
	"sync"
)

type FrameType int

const (
	YUV422Frames FrameType = iota
	RGBFrames
)

type FrameForwarder struct {
	FrameType FrameType
	PixFmt    []uint8
	Width     int
	Height    int

	IsReady bool
	IsStill bool

	LastFrame image.Image

	recycledFrames []image.Image
	sync.Mutex
}

func (f *FrameForwarder) Init(ft FrameType, pf []uint8, width int, height int) {
	f.FrameType = ft
	f.PixFmt = pf
	f.Width = width
	f.Height = height
}

func (f *FrameForwarder) SendFrame(frame image.Image) {
	f.LastFrame = frame
}

func (f *FrameForwarder) GetBlankFrame() image.Image {
	f.Lock()
	defer f.Unlock()

	if len(f.recycledFrames) == 0 {
		switch f.FrameType {
		case YUV422Frames:
			return image.NewYCbCr(image.Rect(0, 0, f.Width, f.Height), image.YCbCrSubsampleRatio422)
		case RGBFrames:
			return image.NewNRGBA(image.Rect(0, 0, f.Width, f.Height))
		default:
			panic("unknown frame type")
		}
	}
	fr := f.recycledFrames[0]
	f.recycledFrames = f.recycledFrames[1:]
	return fr
}

func (f *FrameForwarder) RecycleFrame(frame image.Image) {
	f.Lock()
	defer f.Unlock()
	f.recycledFrames = append(f.recycledFrames, frame)
}
