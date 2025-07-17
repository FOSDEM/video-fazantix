package layer

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/fosdem/fazantix/lib/encdec"
)

type FrameHold int

const (
	NoHold FrameHold = iota
	HoldUpdate
	Hold
)

type FrameForwarder struct {
	encdec.FrameInfo

	Name      string
	Allocator encdec.FrameAllocator

	IsReady bool
	HoldFrame FrameHold

	curReadingFrame *encdec.Frame
	FrameAge        time.Duration

	TextureIDs [3]uint32

	bin []*encdec.Frame
	sync.Mutex

	FramebufferID uint32
	LastFrameID   uint64

	DroppedFramesIn  uint64
	DroppedFramesOut uint64
}

func (f *FrameForwarder) Init(name string, info *encdec.FrameInfo, alloc encdec.FrameAllocator) {
	f.Name = name
	f.Allocator = alloc
	f.FrameInfo = *info
	f.FrameAge = 0
	f.allocateFrames(info.NumAllocatedFrames)
}

func (f *FrameForwarder) GetFrameForReading() *encdec.Frame {
	f.Lock()
	defer f.Unlock()

	if f.HoldFrame == Hold {
		// Don't upload the frame again when holding
		return nil
	}

	frame := f.curReadingFrame
	if !f.IsReady || frame == nil {
		return nil
	}

	frame.NumReaders.Add(1)

	if f.HoldFrame == HoldUpdate {
		// Don't send this frame again on the next request
		f.HoldFrame = Hold
	}
	return frame
}

func (f *FrameForwarder) FinishedReading(frame *encdec.Frame) {
	f.Lock()
	defer f.Unlock()

	numReaders := frame.NumReaders.Add(-1)
	if numReaders < 0 {
		panic("FinishedReading called on frame with no readers")
	}
	if numReaders == 0 && frame.MarkedForRecycling {
		f.recycleFrame(frame)
	}
}

func (f *FrameForwarder) GetFrameForWriting() *encdec.Frame {
	f.Lock()
	defer f.Unlock()

	if len(f.bin) == 0 {
		f.DroppedFramesOut += 1
		return nil
	}

	frame := f.bin[len(f.bin)-1]
	f.bin = f.bin[:len(f.bin)-1]

	f.LastFrameID += 1
	frame.ID = f.LastFrameID

	frame.MarkedForRecycling = false
	return frame
}

func (f *FrameForwarder) FinishedWriting(frame *encdec.Frame) {
	f.Lock()
	defer f.Unlock()

	if f.curReadingFrame != nil {
		if f.curReadingFrame.NumReaders.Load() == 0 {
			f.recycleFrame(f.curReadingFrame)
		} else {
			f.curReadingFrame.MarkedForRecycling = true
		}
	}

	f.curReadingFrame = frame

	f.FrameAge = 0
	f.IsReady = true
	if f.HoldFrame == Hold {
		f.HoldFrame = HoldUpdate
	}
}

func (f *FrameForwarder) FailedWriting(frame *encdec.Frame) {
	f.Lock()
	defer f.Unlock()

	f.DroppedFramesIn += 1

	f.recycleFrame(frame)
}

func (f *FrameForwarder) recycleFrame(frame *encdec.Frame) {
	if len(f.bin) >= cap(f.bin) || cap(f.bin) != f.FrameInfo.NumAllocatedFrames {
		panic("more frames returned than extracted??")
	}
	f.bin = append(f.bin, frame)
}

func (f *FrameForwarder) allocateFrames(num int) {
	if num < 1 {
		return
	}
	f.bin = make([]*encdec.Frame, num)
	for i := range num {
		f.bin[i] = f.Allocator.NewFrame(&f.FrameInfo)
	}
}

func (f *FrameForwarder) Log(msg string, args ...interface{}) {
	log.Printf("[%s] %s\n", f.Name, fmt.Sprintf(msg, args...))
}

func (f *FrameForwarder) Age(dt time.Duration) {
	f.FrameAge += dt
	if f.HoldFrame == NoHold && f.FrameAge > 1*time.Second {
		f.IsReady = false
	}
}
