package layer

import (
	"image"
)

type FrameType int

const (
	YUV422Frames FrameType = iota
	RGBFrames
)

type FrameForwarder struct {
	FrameType FrameType
	PixFmt    []uint8

	LastFrame image.Image
}

func (f *FrameForwarder) Init() {
}

func (f *FrameForwarder) SendFrame(frame image.Image) {
	f.LastFrame = frame
}

func (f *FrameForwarder) GetBlankRGBFrame(width int, height int) *image.NRGBA {
	return image.NewNRGBA(image.Rect(0, 0, width, height))
}

func (f *FrameForwarder) GetBlankYUV422Frame(width int, height int) *image.YCbCr {
	return image.NewYCbCr(image.Rect(0, 0, width, height), image.YCbCrSubsampleRatio422)
}
