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

	outputFramesRGB    chan *image.NRGBA
	outputFramesYUV422 chan *image.YCbCr
}

func (f *FrameForwarder) Init() {
	f.outputFramesRGB = make(chan *image.NRGBA)
	f.outputFramesYUV422 = make(chan *image.YCbCr)
}

func (f *FrameForwarder) GenRGBFrames() <-chan *image.NRGBA {
	return f.outputFramesRGB
}

func (f *FrameForwarder) GenYUV422Frames() <-chan *image.YCbCr {
	return f.outputFramesYUV422
}

func (f *FrameForwarder) SendRGBFrame(frame *image.NRGBA) {
	f.outputFramesRGB <- frame
}

func (f *FrameForwarder) SendYUV422Frame(frame *image.YCbCr) {
	f.outputFramesYUV422 <- frame
}

func (f *FrameForwarder) GetBlankRGBFrame(width int, height int) *image.NRGBA {
	return image.NewNRGBA(image.Rect(0, 0, width, height))
}

func (f *FrameForwarder) GetBlankYUV422Frame(width int, height int) *image.YCbCr {
	return image.NewYCbCr(image.Rect(0, 0, width, height), image.YCbCrSubsampleRatio422)
}
