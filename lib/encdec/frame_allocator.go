package encdec

import (
	"fmt"

	"github.com/fosdem/fazantix/lib/rendering/renderconsts"
)

type SwizzleConfig [4]renderconsts.Color

type FrameCfg struct {
	Width              int
	Height             int
	NumAllocatedFrames int `yaml:"num_allocated_frames"`
}

type FrameInfo struct {
	FrameCfg
	FrameType FrameType
	PixFmt    []uint8
	Swizzle   SwizzleConfig
}

type FrameAllocator interface {
	NewFrame(info *FrameInfo) *Frame
}

type DumbFrameAllocator struct {
	LastID uint32
}

func (d *DumbFrameAllocator) NewFrame(info *FrameInfo) *Frame {
	t := info.FrameType

	n, w, h := calcFrameSize(info)

	f := &Frame{
		Data:   make([]byte, n),
		Width:  w,
		Height: h,
		Type:   t,
		ID:     0,
		SoulID: d.LastID,
	}
	d.LastID += 1

	return f
}

type FixedFrameAllocator struct {
	LastID uint32
	getBuf func(uint32) []byte
}

func NewFixedFrameAllocator(getBuf func(uint32) []byte) *FixedFrameAllocator {
	return &FixedFrameAllocator{getBuf: getBuf}
}

func (d *FixedFrameAllocator) NewFrame(info *FrameInfo) *Frame {
	t := info.FrameType

	_, w, h := calcFrameSize(info)

	f := &Frame{
		Data:   d.getBuf(d.LastID),
		Width:  w,
		Height: h,
		Type:   t,
		ID:     0,
		SoulID: d.LastID,
	}
	d.LastID += 1

	return f
}

// NullFrameAllocator allocates frames without any data buffer,
// such that the writer is supposed to take care of providing the buffer memory
type NullFrameAllocator struct {
	LastID uint32
}

func (n *NullFrameAllocator) NewFrame(info *FrameInfo) *Frame {
	t := info.FrameType

	_, w, h := calcFrameSize(info)

	f := &Frame{
		Data:   []byte{},
		Width:  w,
		Height: h,
		Type:   t,
		ID:     0,
		SoulID: n.LastID,
	}
	n.LastID += 1

	return f
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

func calcFrameSize(info *FrameInfo) (int, int, int) {
	t := info.FrameType
	w := info.Width
	h := info.Height

	switch t {
	case YUV422Frames:
		return w * h * 2, w, h
	case YUV422pFrames:
		return w * h * 2, w, h
	case RGBAFrames:
		fallthrough
	case BGRAFrames:
		return w * h * 4, w, h
	case RGBFrames:
		return w * h * 3, w, h
	default:
		panic("unknown frame type")
	}
}
