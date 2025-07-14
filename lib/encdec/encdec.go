package encdec

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"log"

	"github.com/go-gl/gl/v4.1-core/gl"
)

type FrameType int

const (
	YUV422Frames FrameType = iota
	RGBAFrames
	RGBFrames
)

type PixelBufferStatus int

const (
	PixelBufferUninitialized PixelBufferStatus = iota
	PixelBufferPack
	PixelBufferUnpack
	PixelBufferUnmapped
)

type Frame struct {
	Data           []byte
	TextureOffsets [3][2]int
	NumTextures    int
	TextureWidths  [3]int
	TextureHeights [3]int
	Width          int
	Height         int
	LastOffset     int
	Type           FrameType
	BufferStatus   PixelBufferStatus
	Buffers        []uint32
}

func (i *Frame) Clear() {
	i.NumTextures = 0
	i.LastOffset = 0
}

func (i *Frame) MakeTexture(n int, w int, h int) []uint8 {
	newOffset := i.LastOffset + n
	i.TextureOffsets[i.NumTextures][0] = i.LastOffset
	i.TextureOffsets[i.NumTextures][1] = newOffset
	i.TextureWidths[i.NumTextures] = w
	i.TextureHeights[i.NumTextures] = h
	i.NumTextures++

	texture := i.Data[i.LastOffset:newOffset]

	i.LastOffset = newOffset
	return texture
}

func clearOpenGLError() {
	for {
		glerr := gl.GetError()
		if glerr == 0 {
			return
		}
	}
}

func HandleOpenGLError() {
	glerr := gl.GetError()
	if glerr == gl.NO_ERROR {
		return
	}
	log.Fatalf("OpenGL Error %d\n", glerr)
}

func (i *Frame) AllocatePBO() {
	if i.BufferStatus != PixelBufferUninitialized {
		panic("Double allocation")
	}
	if i.NumTextures == 0 {
		i.NumTextures = 1
	}
	i.Buffers = make([]uint32, i.NumTextures)
	gl.GenBuffers(int32(i.NumTextures), &i.Buffers[0])
	HandleOpenGLError()
	i.BufferStatus = PixelBufferUnmapped
}

func (i *Frame) PBOToTexture(textureIDs [3]uint32) {
	if i.BufferStatus != PixelBufferUnmapped {
		panic("Attempting to pack into PBO of wrong state")
	}
	i.BufferStatus = PixelBufferUnpack
	channelType := uint32(gl.RED)
	if i.Type == RGBAFrames {
		channelType = gl.RGBA
	}

	for idx := range i.NumTextures {
		gl.BindTexture(gl.TEXTURE_2D, textureIDs[idx])
		gl.BindBuffer(gl.PIXEL_PACK_BUFFER, i.Buffers[idx])
		gl.TexSubImage2D(gl.TEXTURE_2D, 0, 0, 0, int32(i.TextureWidths[idx]), int32(i.TextureHeights[idx]), channelType, gl.UNSIGNED_BYTE, gl.Ptr(0))
		gl.BindBuffer(gl.PIXEL_UNPACK_BUFFER, 0)
	}
	i.BufferStatus = PixelBufferUnmapped
}

func (i *Frame) FrameToPBO() {
	if i.BufferStatus != PixelBufferUnmapped {
		panic("Attempting to pack into PBO of wrong state")
	}
	for idx := range i.NumTextures {
		dataPtr, _, _ := i.Texture(idx)
		gl.BindBuffer(gl.PIXEL_PACK_BUFFER, i.Buffers[idx])
		gl.BufferData(gl.PIXEL_UNPACK_BUFFER, len(dataPtr), gl.Ptr(dataPtr), gl.STREAM_DRAW)
		bufferPtr := gl.MapBuffer(gl.PIXEL_UNPACK_BUFFER, gl.WRITE_ONLY)

		// TODO: Copy dataPtr ([]uint8) to bufferPtr (unsafe.Pointer)

		// Unmapbuffer should accept a gl.Ptr instead of an uint32
		bufferPtrBug := (*uint32)(bufferPtr)
		gl.UnmapBuffer(*bufferPtrBug)
		gl.BindBuffer(gl.PIXEL_UNPACK_BUFFER, 0)
	}
}

func (i *Frame) FramebufferToPBO() {
	if i.BufferStatus == PixelBufferUninitialized {
		i.AllocatePBO()
		gl.BindBuffer(gl.PIXEL_PACK_BUFFER, i.Buffers[0])
		HandleOpenGLError()
		gl.BufferData(gl.PIXEL_PACK_BUFFER, i.Width*i.Height*4, gl.Ptr(nil), gl.STREAM_READ)
		HandleOpenGLError()

	}
	if i.BufferStatus != PixelBufferUnmapped {
		panic("Attempting to pack into PBO of wrong state")
	}
	gl.BindBuffer(gl.PIXEL_PACK_BUFFER, i.Buffers[0])
	HandleOpenGLError()
	gl.ReadPixels(0, 0, int32(i.Width), int32(i.Height), gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(nil))
	HandleOpenGLError()

	gl.BindBuffer(gl.PIXEL_PACK_BUFFER, 0)
	HandleOpenGLError()
}

func (i *Frame) PBOToFrame() {
	if i.BufferStatus != PixelBufferUnmapped {
		panic("Attempting to pack into PBO of wrong state")
	}

	gl.BindBuffer(88, i.Buffers[0])
	HandleOpenGLError()

	bufferPtr := gl.MapBuffer(77, gl.READ_ONLY)
	HandleOpenGLError()

	if bufferPtr != nil {
		fmt.Println("FRAME")
		buffer := *(*[]uint8)(bufferPtr)

		i.Clear()
		framePtr := i.MakeTexture(i.Width*i.Height*4, i.Width, i.Height)

		copy(framePtr, buffer)
		gl.UnmapBuffer(gl.PIXEL_PACK_BUFFER)
	}

	gl.BindBuffer(gl.PIXEL_PACK_BUFFER, 0)
	HandleOpenGLError()

}

func (i *Frame) Texture(idx int) ([]byte, int, int) {
	start := i.TextureOffsets[idx][0]
	upto := i.TextureOffsets[idx][1]

	ptr := i.Data[start:upto]
	w := i.TextureWidths[idx]
	h := i.TextureHeights[idx]
	return ptr, w, h
}

func DecodeYUYV422(buf []byte, into *Frame) error {
	into.Clear()

	numPixels := len(buf) / 2
	numChromaPixels := numPixels / 2 // the chroma plane is half-sized

	if len(buf) != len(into.Data) {
		return fmt.Errorf("expected buffer of size %d but got %d", len(into.Data), len(buf))
	}

	Y := into.MakeTexture(numPixels, into.Width, into.Height)
	U := into.MakeTexture(numChromaPixels, into.Width/2, into.Height)
	V := into.MakeTexture(numChromaPixels, into.Width/2, into.Height)

	for i := range U {
		j := i * 4
		Y[i*2] = buf[j]
		U[i] = buf[j+1]
		Y[i*2+1] = buf[j+2]
		V[i] = buf[j+3]
	}
	return nil
}

func PrepareYUYV422p(into *Frame) error {
	into.Clear()

	numPixels := into.Width * into.Height
	numChromaPixels := numPixels / 2 // the chroma plane is half-sized

	into.MakeTexture(numPixels, into.Width, into.Height)
	into.MakeTexture(numChromaPixels, into.Width/2, into.Height)
	into.MakeTexture(numChromaPixels, into.Width/2, into.Height)

	return nil
}

func DecodeRGBfromImage(buf []byte, into *Frame) error {
	into.Clear()

	img, _, err := image.Decode(bytes.NewReader(buf))
	if err != nil {
		return err
	}

	return FrameFromImage(img, into)
}

func FrameFromImage(img image.Image, into *Frame) error {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()

	if w != into.Width || h != into.Height {
		return fmt.Errorf("expected image of size %dx%d but got %dx%d", into.Width, into.Height, w, h)
	}

	nrgba := image.NewNRGBA(image.Rect(0, 0, w, h))
	bufSize := len(nrgba.Pix)

	if bufSize != len(into.Data) {
		return fmt.Errorf("expected buffer of size %d but got %d", len(into.Data), bufSize)
	}

	nrgba.Pix = into.MakeTexture(bufSize, into.Width, into.Height)
	draw.Draw(nrgba, nrgba.Bounds(), img, img.Bounds().Min, draw.Src)
	return nil
}

func (f FrameType) String() string {
	switch f {
	case YUV422Frames:
		return "YUV422"
	case RGBAFrames:
		return "RGBA"
	default:
		panic("unknown frame type")
	}
}
