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
}

type FrameAllocator interface {
	NewFrame(info *FrameInfo) *Frame
}

type DumbFrameAllocator struct{}

func (d *DumbFrameAllocator) NewFrame(info *FrameInfo) *Frame {
	t := info.FrameType
	w := info.Width
	h := info.Height

	switch t {
	case YUV422Frames:
		return d.makeFrame(t, w*h*2, w, h)
	case RGBAFrames:
		return d.makeFrame(t, w*h*4, w, h)
	case RGBFrames:
		return d.makeFrame(t, w*h*3, w, h)
	default:
		panic("unknown frame type")
	}
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
