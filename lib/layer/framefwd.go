package layer

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/rendering"
)

type FrameForwarder struct {
	encdec.FrameInfo

	Name      string
	Allocator encdec.FrameAllocator

	IsReady bool
	IsStill bool

	lastFrame *encdec.Frame
	FrameAge  time.Duration

	TextureIDs [3]uint32

	bin []*encdec.Frame
	sync.Mutex

	FramebufferID uint32
}

func (f *FrameForwarder) Init(name string, info *encdec.FrameInfo, alloc encdec.FrameAllocator) {
	f.Name = name
	f.Allocator = alloc
	f.FrameInfo = *info
	f.FrameAge = 0
	f.allocateFrames(info.NumAllocatedFrames)
}

func (f *FrameForwarder) GetFrameForReading() *encdec.Frame {
	if !f.IsReady {
		return nil
	}
	return f.lastFrame
}

func (f *FrameForwarder) FinishedReading(frame *encdec.Frame) {
	// TODO: implement
}

func (f *FrameForwarder) SendFrame(frame *encdec.Frame) {
	oldLastFrame := f.lastFrame
	f.lastFrame = frame
	f.FrameAge = 0
	f.IsReady = true
	if oldLastFrame != nil {
		f.recycleFrame(oldLastFrame)
	}
}

func (f *FrameForwarder) GetBlankFrame() *encdec.Frame {
	f.Lock()
	defer f.Unlock()

	if len(f.bin) == 0 {
		// TODO: log a framedrop
		return nil
	}
	fr := f.bin[len(f.bin)-1]
	f.bin = f.bin[:len(f.bin)-1]
	return fr
}

func (f *FrameForwarder) recycleFrame(frame *encdec.Frame) {
	f.Lock()
	defer f.Unlock()
	if len(f.bin) >= cap(f.bin) || cap(f.bin) != f.FrameInfo.NumAllocatedFrames {
		panic("more frames returned than extracted??")
	}
	f.bin = append(f.bin, frame)
}

func (f *FrameForwarder) allocateFrames(num int) {
	if num < 1 {
		panic(fmt.Sprintf("[%s]: %d is an invalid number of requested allocated frames", f.Name, num))
	}
	f.bin = make([]*encdec.Frame, num)
	for i := range num {
		f.bin[i] = f.Allocator.NewFrame(&f.FrameInfo)
	}
}

func (f *FrameForwarder) Log(msg string, args ...interface{}) {
	log.Printf("[%s] %s\n", f.Name, fmt.Sprintf(msg, args...))
}

func (f *FrameForwarder) SetupTextures() {
	width := f.Width
	height := f.Height

	switch f.FrameType {
	case encdec.YUV422Frames:
		f.TextureIDs[0] = rendering.SetupYUVTexture(width, height)
		f.TextureIDs[1] = rendering.SetupYUVTexture(width/2, height)
		f.TextureIDs[2] = rendering.SetupYUVTexture(width/2, height)
	case encdec.RGBAFrames:
		f.TextureIDs[0] = rendering.SetupRGBATexture(width, height)
	case encdec.RGBFrames:
		f.TextureIDs[0] = rendering.SetupRGBTexture(width, height)
	}
}

func (f *FrameForwarder) UseAsFramebuffer() {
	if f.FrameType != encdec.RGBFrames {
		panic("trying to use a non-rgb frame forwarder as a framebuffer")
	}
	f.FramebufferID = rendering.UseTextureAsFramebuffer(f.TextureIDs[0])
}

func (f *FrameForwarder) Age(dt time.Duration) {
	f.FrameAge += dt
	if !f.IsStill && f.FrameAge > 1*time.Second {
		f.IsReady = false
	}
}
