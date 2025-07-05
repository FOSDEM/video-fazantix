package encdec

type FrameInfo struct {
	FrameType FrameType
	PixFmt    []uint8
	Width     int
	Height    int
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
