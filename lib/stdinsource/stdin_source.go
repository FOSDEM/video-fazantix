package stdinsource

import (
	"bufio"
	"io"
	"os"
	"time"

	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/layer"
)

type StdinSource struct {
	Width     int
	Height    int
	Rate      int
	frameSize int
	frames    layer.FrameForwarder
	reader    *bufio.Reader
}

func New(width, height, rate int, alloc encdec.FrameAllocator) *StdinSource {
	f := &StdinSource{Width: width, Height: height, frameSize: width * height * 3}
	frameCfg := &encdec.FrameCfg{
		Width:              width,
		Height:             height,
		NumAllocatedFrames: 5,
	}
	f.Rate = rate
	f.frames.Init(
		"stdin",
		&encdec.FrameInfo{
			FrameType: encdec.RGBFrames,
			PixFmt:    []uint8{},
			FrameCfg:  *frameCfg,
		},
		alloc,
	)
	return f
}

func (f *StdinSource) Start() bool {
	f.reader = bufio.NewReaderSize(os.Stdin, f.frameSize)
	go f.processStdin()
	return true
}

func (f *StdinSource) processStdin() {
	ftime := time.Now()
	frameTime := (1000000 / time.Duration(f.Rate)) * time.Microsecond
	for {
		frame := f.frames.GetFrameForWriting()
		frame.Clear()
		frame.MakeTexture(f.frameSize, f.Width, f.Height)
		past := time.Since(ftime)
		time.Sleep(frameTime - past)
		ftime = time.Now()
		_, err := io.ReadFull(f.reader, frame.Data)
		if err != nil {
			f.log("could not read from stdin: %s", err)
			f.frames.FailedWriting(frame)
			return
		}

		f.frames.FinishedWriting(frame)
	}
}

func (f *StdinSource) Frames() *layer.FrameForwarder {
	return &f.frames
}

func (f *StdinSource) log(msg string, args ...interface{}) {
	f.Frames().Log(msg, args...)
}
