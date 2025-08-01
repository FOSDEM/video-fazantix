package encdec

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"sync/atomic"
)

type FrameType int

const (
	YUV422Frames FrameType = iota
	YUV422pFrames
	RGBAFrames
	RGBFrames
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

	NumReaders         atomic.Int32
	MarkedForRecycling bool
	ID                 uint64
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

func PrepareYUYV(into *Frame) error {
	into.Clear()

	numPixels := into.Width * into.Height / 2

	into.MakeTexture(numPixels, into.Width/2, into.Height)

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
	into.Clear()
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
	case YUV422pFrames:
		return "YUYV"
	case RGBAFrames:
		return "RGBA"
	case RGBFrames:
		return "RGB"
	default:
		panic("unknown frame type")
	}
}
