package main

import (
	"log"
	"os"
	"context"
	"fmt"
	"bytes"

	"image"
	"image/draw"
	_ "image/png"
	_ "image/jpeg"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"

		"github.com/vladimirvivien/go4vl/device"
	"github.com/vladimirvivien/go4vl/v4l2"
)

type Coordinate struct {
	x float32
	y float32
}

type Mask struct {
	top float32
	bottom float32
	left float32
	right float32
}

type Source struct {
	Name string
	IsStill bool

	IsVisible bool
	IsReady bool

	Size Coordinate
	Position Coordinate
	Mask Mask

	Texture uint32

	Width uint32
	Height uint32
	Squeeze float32

	Frames <-chan []byte
	Images chan *image.NRGBA
}

func newSource(name string) Source {
	s := Source{Name: name, IsStill: true, IsVisible: false, IsReady: false}
	s.Size = Coordinate{x: 1.0, y:1.0}
	s.Squeeze = 1.0
	s.Position = Coordinate{x: 0.5, y: 0.5}
	s.Mask = Mask{top: 0, bottom: 0, left: 0, right: 0}
	return s
}

func (s *Source) Move(x float32, y float32, size float32) {
	s.Position.x = x
	s.Position.y = y
	s.Size.x = size
	s.Size.y = size / s.Squeeze
}

func (s *Source) LoadStill(path string) bool {
	log.Printf("[%s] Loading %s", s.Name, path)
	imgFile, err := os.Open(path)
	if err != nil {
		log.Printf("[%s] Error: %s", s.Name, err)
		return false
	}

	img, _, err := image.Decode(imgFile)
	if err != nil {
		log.Printf("[%s] Error: %s", s.Name, err)
		return false
	}

	rgba := image.NewRGBA(img.Bounds())
	if rgba.Stride != rgba.Rect.Size().X*4 {
		log.Printf("[%s] Unsupported stride", s.Name)
		return false
	}
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Src)
	log.Printf("[%s] Size: %dx%d", s.Name, rgba.Bounds().Dx(), rgba.Bounds().Dy())

	gl.GenTextures(1, &s.Texture)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, s.Texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

	borderColor := mgl32.Vec4{0, 0, 0, 0}
	gl.TexParameterfv(gl.TEXTURE_2D, gl.TEXTURE_BORDER_COLOR, &borderColor[0])
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_BORDER)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_BORDER)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(rgba.Rect.Size().X),
		int32(rgba.Rect.Size().Y),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(rgba.Pix))

	s.IsReady = true
	return true
}

func (s *Source) DecodeFramesJPEG() {
	var frame []byte
	for frame = range s.Frames {
		img, _, err := image.Decode(bytes.NewReader(frame))
		if err != nil {
			fmt.Printf("[%s] decode failure: %s", s.Name, err)
		}
		bounds := img.Bounds()
		nrgba := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
		draw.Draw(nrgba, nrgba.Bounds(), img, bounds.Min, draw.Src)
		s.IsReady = true
		s.Images <- nrgba
	}
}

func (s *Source) LoadV4l2(devName string, width int, height int) bool {
	log.Printf("[%s] Loading v4l2 device %s", s.Name, devName)
	s.IsStill = false
	camera, err := device.Open(
		devName,
		device.WithPixFormat(v4l2.PixFormat{PixelFormat: v4l2.PixelFmtMJPEG, Width: uint32(width), Height: uint32(height)}),
	)
	if err != nil {
		log.Printf("[%s] Failed to open device: %s", s.Name, err)
		return false
	}
	// defer camera.Close()
	log.Printf("[%s] Opened device at %dx%d", s.Name, width, height)

	gl.GenTextures(1, &s.Texture)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, s.Texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

	borderColor := mgl32.Vec4{0, 0, 0, 0}
	gl.TexParameterfv(gl.TEXTURE_2D, gl.TEXTURE_BORDER_COLOR, &borderColor[0])
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_BORDER)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_BORDER)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(width),
		int32(height),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		nil)

	s.Squeeze = (float32(width)/float32(height)) / (16.0/9.0)
	if err := camera.Start(context.TODO()); err != nil {
		log.Fatalf("[%s] camera start: %s", s.Name, err)
	}
	s.Frames = camera.GetOutput()
	s.Images = make(chan *image.NRGBA)
	go s.DecodeFramesJPEG()
	return true
}
