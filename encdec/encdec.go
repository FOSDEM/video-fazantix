package encdec

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
)

type FrameType int

const (
	YUV422Frames FrameType = iota
	RGBAFrames
	RGBFrames
)

type ImageData struct {
	Data           []byte
	TextureOffsets [3][2]int
	NumTextures    int
	TextureWidths  [3]int
	TextureHeights [3]int
	Width          int
	Height         int
	LastOffset     int
	Type           FrameType
}

func NewFrame(t FrameType, w int, h int) *ImageData {
	switch t {
	case YUV422Frames:
		return makeFrame(t, w*h*2, w, h)
	case RGBAFrames:
		return makeFrame(t, w*h*4, w, h)
	case RGBFrames:
		return makeFrame(t, w*h*3, w, h)
	default:
		panic("unknown frame type")
	}
}

func (i *ImageData) Clear() {
	i.NumTextures = 0
	i.LastOffset = 0
}

func makeFrame(t FrameType, n int, w int, h int) *ImageData {
	return &ImageData{
		Data:   make([]byte, n),
		Width:  w,
		Height: h,
		Type:   t,
	}
}

func (i *ImageData) MakeTexture(n int, w int, h int) []uint8 {
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

func (i *ImageData) Texture(idx int) ([]byte, int, int) {
	start := i.TextureOffsets[idx][0]
	upto := i.TextureOffsets[idx][1]

	ptr := i.Data[start:upto]
	w := i.TextureWidths[idx]
	h := i.TextureHeights[idx]
	return ptr, w, h
}

func DecodeYUYV422(buf []byte, into *ImageData) error {
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

func PrepareYUYV422p(into *ImageData) error {
	into.Clear()

	numPixels := into.Width * into.Height
	numChromaPixels := numPixels / 2 // the chroma plane is half-sized

	into.MakeTexture(numPixels, into.Width, into.Height)
	into.MakeTexture(numChromaPixels, into.Width/2, into.Height)
	into.MakeTexture(numChromaPixels, into.Width/2, into.Height)

	return nil
}

func DecodeRGBfromImage(buf []byte, into *ImageData) error {
	into.Clear()

	img, _, err := image.Decode(bytes.NewReader(buf))
	if err != nil {
		return err
	}

	return FrameFromImage(img, into)
}

func FrameFromImage(img image.Image, into *ImageData) error {
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
