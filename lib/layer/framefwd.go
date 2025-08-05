package layer

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/metrics"
)

type FrameHold int

const (
	NoHold FrameHold = iota
	HoldUpdate
	Hold
)

// FrameForwarder takes care of synchronising frames between a single
// writer and multiple readers.
// It is suitable for streaming, not recording, because it is designed
// to drop unused source frames instead of queueing them, thus achieving
// minimal latency.
// TODO: implement another FrameForwarder that works in latency-agnostic
// queueing mode and let the user choose to use it when they use fazantix for
// recording instead of streaming.
type FrameForwarder struct {
	encdec.FrameInfo

	Name      string
	Allocator encdec.FrameAllocator

	IsReady   bool
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

	metrics metrics.StreamMetrics
}

func (f *FrameForwarder) Init(name string, info *encdec.FrameInfo, alloc encdec.FrameAllocator) {
	f.Name = name
	f.Allocator = alloc
	f.FrameInfo = *info
	f.FrameAge = 0
	f.allocateFrames(info.NumAllocatedFrames)
	f.metrics = metrics.NewStreamMetrics(name)
}

// GetFrameForReading gets the latest fully-written frame and blocks
// the writer from using it. Multiple readers can get the same frame
// concurrently, and the frame is released as available for writing
// into only after all readers have released it.
// Users must ensure that NumAllocatedFrames is big enough for cases
// when some readers are slower than others and hold older frames
// for reading for long enough that those frames are still unavailable
// during the next writing cycle.
// For each call of GetFrameForReading() there should be a corresponding
// call of exactly one of FinishedReading() or FailedReading().
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

func (f *FrameForwarder) GetAnyFrameForReading() *encdec.Frame {
	f.Lock()
	defer f.Unlock()

	frame := f.curReadingFrame
	if !f.IsReady || frame == nil {
		return nil
	}

	frame.NumReaders.Add(1)

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

// GetFrameForWriting gets an unused frame for writing into.
// The writer may call GetFrameForWriting() multiple times to get multiple
// frames for writing, but they have to ensure that there are enough frames
// left in the pool for readers to hold.
// For each call of GetFrameForWriting() there should be exactly one corresponding
// call of either FinishedWriting() or FailedWriting() to put the frame back into
// the pool. Failure to do so will result in frames leaking and eventually a panic
// when the pool becomes empty.
func (f *FrameForwarder) GetFrameForWriting() *encdec.Frame {
	f.Lock()
	defer f.Unlock()

	if len(f.bin) == 0 {
		f.DroppedFramesOut += 1
		f.metrics.FramesDropped.Inc()
		return nil
	}

	frame := f.bin[len(f.bin)-1]
	f.bin = f.bin[:len(f.bin)-1]

	f.LastFrameID += 1
	frame.ID = f.LastFrameID

	frame.MarkedForRecycling = false
	return frame
}

// FinishedWriting sets the given frame as a "latest frame", so that
// new readers will use that frame. The previous "latest frame" is
// put back into the pool of frames that can be taken out with
// GetFrameForWriting()
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
	f.metrics.FramesForwarded.Inc()

	f.FrameAge = 0
	f.IsReady = true
	if f.HoldFrame == Hold {
		f.HoldFrame = HoldUpdate
	}
}

// FailedWriting puts a frame back into the pool without updating
// the latest frame pointer
func (f *FrameForwarder) FailedWriting(frame *encdec.Frame) {
	f.Lock()
	defer f.Unlock()

	f.DroppedFramesIn += 1
	f.metrics.FramesDropped.Inc()

	f.recycleFrame(frame)
}

func (f *FrameForwarder) AvailableFramesForWriting() int {
	return len(f.bin)
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

func (f *FrameForwarder) Debug(msg string, args ...interface{}) {
	// TODO: Implement debug loglevel
	log.Printf("[%s] %s\n", f.Name, fmt.Sprintf(msg, args...))
}

func (f *FrameForwarder) Error(msg string, args ...interface{}) {
	// TODO: Implement error loglevel
	log.Printf("[%s] %s\n", f.Name, fmt.Sprintf(msg, args...))
}

func (f *FrameForwarder) Age(dt time.Duration) {
	f.FrameAge += dt
	if f.HoldFrame == NoHold && f.FrameAge > 1*time.Second {
		f.IsReady = false
	}
}
