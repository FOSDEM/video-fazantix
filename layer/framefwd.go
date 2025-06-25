package layer

import (
	"fmt"
	"log"
	"sync"

	"github.com/fosdem/fazantix/encdec"
	"github.com/fosdem/fazantix/rendering"
)

type FrameForwarder struct {
	FrameType encdec.FrameType
	PixFmt    []uint8
	Width     int
	Height    int

	Name    string
	IsReady bool
	IsStill bool

	LastFrame *encdec.ImageData
	FrameAge int

	TextureIDs [3]uint32

	recycledFrames []*encdec.ImageData
	sync.Mutex

	FramebufferID uint32
}

func (f *FrameForwarder) Init(name string, ft encdec.FrameType, pf []uint8, width int, height int) {
	f.Name = name
	f.FrameType = ft
	f.PixFmt = pf
	f.Width = width
	f.Height = height
	f.FrameAge = 0
}

func (f *FrameForwarder) SendFrame(frame *encdec.ImageData) {
	oldLastFrame := f.LastFrame
	f.LastFrame = frame
	f.FrameAge = 0
	if oldLastFrame != nil {
		f.recycleFrame(oldLastFrame)
	}
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

func (f *FrameForwarder) recycleFrame(frame *encdec.ImageData) {
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
