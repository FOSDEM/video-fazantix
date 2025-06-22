package layer

import (
	"sync"

	"github.com/fosdem/fazantix/encdec"
)

type FrameForwarder struct {
	FrameType encdec.FrameType
	PixFmt    []uint8
	Width     int
	Height    int

	IsReady bool
	IsStill bool

	LastFrame *encdec.ImageData

	recycledFrames []*encdec.ImageData
	sync.Mutex
}

func (f *FrameForwarder) Init(ft encdec.FrameType, pf []uint8, width int, height int) {
	f.FrameType = ft
	f.PixFmt = pf
	f.Width = width
	f.Height = height
}

func (f *FrameForwarder) SendFrame(frame *encdec.ImageData) {
	f.LastFrame = frame
}

func (f *FrameForwarder) GetBlankFrame() *encdec.ImageData {
	f.Lock()
	defer f.Unlock()

	if len(f.recycledFrames) == 0 {
		return encdec.NewFrame(f.FrameType, f.Width, f.Height)
	}
	fr := f.recycledFrames[0]
	f.recycledFrames = f.recycledFrames[1:]
	return fr
}

func (f *FrameForwarder) RecycleFrame(frame *encdec.ImageData) {
	f.Lock()
	defer f.Unlock()
	f.recycledFrames = append(f.recycledFrames, frame)
}
