package encdec

import "fmt"

type FrameCfg struct {
	Width              int
	Height             int
	NumAllocatedFrames int `yaml:"num_allocated_frames"`
}

type FrameInfo struct {
	FrameCfg
	FrameType FrameType
	PixFmt    []uint8
	TransportType
}

type FrameAllocator interface {
	NewFrame(info *FrameInfo) *Frame
}

type DumbFrameAllocator struct{}

func (d *DumbFrameAllocator) NewFrame(info *FrameInfo) *Frame {
	t := info.FrameType
	w := info.Width
	h := info.Height

	return d.makeFrame(t, info.CalcBufSize(), w, h)
}

func (d *DumbFrameAllocator) makeFrame(t FrameType, n int, w int, h int) *Frame {
	return &Frame{
		Data:   make([]byte, n),
		Width:  w,
		Height: h,
		Type:   t,
	}
}

func (f *FrameCfg) Validate(isWindow bool) error {
	if !isWindow && f.NumAllocatedFrames < 1 {
		return fmt.Errorf("number of allocated frames must be at least 1")
	}
	if isWindow && f.NumAllocatedFrames != 0 {
		return fmt.Errorf("number of allocated frames for window sinks must be 0")
	}
	if f.Width < 1 {
		return fmt.Errorf("width must be at least 1")
	}
	if f.Height < 1 {
		return fmt.Errorf("height must be at least 1")
	}
	return nil
}

func (f *FrameInfo) CalcBufSize() int {
	t := f.FrameType
	w := f.Width
	h := f.Height

	switch t {
	case YUV422Frames:
		return w * h * 2
	case RGBAFrames:
		return w * h * 4
	case RGBFrames:
		return w * h * 3
	default:
		panic("unknown frame type")
	}
}
