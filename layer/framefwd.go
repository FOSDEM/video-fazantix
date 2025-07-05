package layer

import (
	"fmt"
	"log"
	"sync"

	"github.com/fosdem/fazantix/encdec"
	"github.com/fosdem/fazantix/rendering"
)

type FrameForwarder struct {
	encdec.FrameInfo

	Name      string
	Allocator encdec.FrameAllocator

	IsReady bool
	IsStill bool

	LastFrame *encdec.Frame
	FrameAge  int

	TextureIDs [3]uint32

	recycledFrames []*encdec.Frame
	sync.Mutex

	FramebufferID uint32
}

func (f *FrameForwarder) Init(name string, info *encdec.FrameInfo, alloc encdec.FrameAllocator) {
	f.Name = name
	f.Allocator = alloc
	f.FrameInfo = *info
	f.FrameAge = 0
}

func (f *FrameForwarder) SendFrame(frame *encdec.Frame) {
	oldLastFrame := f.LastFrame
	f.LastFrame = frame
	f.FrameAge = 0
	if oldLastFrame != nil {
		f.recycleFrame(oldLastFrame)
	}
}

func (f *FrameForwarder) GetBlankFrame() *encdec.Frame {
	f.Lock()
	defer f.Unlock()

	if len(f.recycledFrames) == 0 {
		return f.Allocator.NewFrame(&f.FrameInfo)
	}
	fr := f.recycledFrames[0]
	f.recycledFrames = f.recycledFrames[1:]
	return fr
}

func (f *FrameForwarder) recycleFrame(frame *encdec.Frame) {
	f.Lock()
	defer f.Unlock()
	f.recycledFrames = append(f.recycledFrames, frame)
}

func (f *FrameForwarder) Log(msg string, args ...interface{}) {
	log.Printf("[%s]: %s\n", f.Name, fmt.Sprintf(msg, args...))
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
