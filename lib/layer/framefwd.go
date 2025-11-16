package layer

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/metrics"
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

	IsReady bool

	curReadingFrame *encdec.Frame
	FrameAge        time.Duration

	TextureIDs [3]uint32

	bin []*encdec.Frame
	sync.Mutex
	frameBarrier *sync.Cond

	HoldFrame          bool
	FramebufferID      uint32
	LastWrittenFrameID uint64
	LastReadFrameID    uint64

	DroppedFramesIn  uint64
	DroppedFramesOut uint64

	metrics metrics.StreamMetrics
	logger  *slog.Logger
}

func (f *FrameForwarder) Init(name string, info *encdec.FrameInfo, alloc encdec.FrameAllocator) {
	f.Name = name
	f.Allocator = alloc
	f.FrameInfo = *info
	f.FrameAge = 0
	f.frameBarrier = sync.NewCond(&f.Mutex)
	f.allocateFrames(info.NumAllocatedFrames)
	f.metrics = metrics.NewStreamMetrics(name)
	f.InitLogging()
}

// GetFreshFrameForReading gets the latest fully-written frame and blocks
// the writer from using it. Multiple readers can get the same frame
// concurrently, and the frame is released as available for writing
// into only after all readers have released it.
// Users must ensure that NumAllocatedFrames is big enough for cases
// when some readers are slower than others and hold older frames
// for reading for long enough that those frames are still unavailable
// during the next writing cycle.
// For each call of GetFrameForReading() there should be a corresponding
// call of exactly one of FinishedReading() or FailedReading().
func (f *FrameForwarder) GetFreshFrameForReading() *encdec.Frame {
	f.Lock()
	defer f.Unlock()
	f.metrics.FramesRequested.Inc()

	frame := f.curReadingFrame
	if !f.IsReady || frame == nil || frame.ID <= f.LastReadFrameID {
		return nil
	}
	f.LastReadFrameID = frame.ID

	frame.NumReaders.Add(1)

	return frame
}

func (f *FrameForwarder) BlockingGetFrameForReading() *encdec.Frame {
	f.Lock()
	defer f.Unlock()
	f.metrics.FramesRequested.Inc()

	var frame *encdec.Frame
	for {
		frame = f.curReadingFrame
		if !f.IsReady || frame == nil || frame.ID <= f.LastReadFrameID {
			f.frameBarrier.Wait()
		} else {
			break
		}
	}

	f.LastReadFrameID = frame.ID

	frame.NumReaders.Add(1)

	return frame
}

func (f *FrameForwarder) GetAnyFrameForReading() *encdec.Frame {
	f.Lock()
	defer f.Unlock()
	f.metrics.FramesRequested.Inc()

	frame := f.curReadingFrame
	if !f.IsReady || frame == nil {
		return nil
	}

	frame.NumReaders.Add(1)

	return frame
}

func (f *FrameForwarder) FinishedReading(frame *encdec.Frame) {
	f.metrics.FramesRead.Inc()

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

	f.LastWrittenFrameID += 1
	frame.ID = f.LastWrittenFrameID

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
	f.metrics.FramesWritten.Inc()

	f.FrameAge = 0
	f.IsReady = true
	f.frameBarrier.Broadcast()
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

func (f *FrameForwarder) InitLogging() {
	logger := slog.Default()
	f.logger = logger.With(slog.String("module", f.Name))
}

func (f *FrameForwarder) Log(msg string, args ...interface{}) {
	f.logger.Info(fmt.Sprintf(msg, args...))
}

func (f *FrameForwarder) Debug(msg string, args ...interface{}) {
	f.logger.Debug(fmt.Sprintf(msg, args...))
}

func (f *FrameForwarder) Error(msg string, args ...interface{}) {
	f.logger.Error(fmt.Sprintf(msg, args...))
}

func (f *FrameForwarder) Age(dt time.Duration) {
	f.FrameAge += dt
	if !f.HoldFrame && f.FrameAge > 1*time.Second {
		f.IsReady = false
	}
}
