package rendering

import (
	"time"
	"unsafe"

	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/layer"
	"github.com/go-gl/gl/v4.1-core/gl"
)

func SendFrameToGPU(frame *encdec.Frame, textureIDs [3]uint32, offset int) {
	channelType := uint32(gl.RED)
	if frame.Type == encdec.RGBAFrames {
		channelType = gl.RGBA
	}

	gl.BindBuffer(frame.GLPixelBufferType, frame.GLPixelBufferID)

	// this buffer has been written to until now, unmap it so that we can read from it
	gl.UnmapBuffer(frame.GLPixelBufferType)

	for j := 0; j < frame.NumTextures; j++ {
		pboOffset, size, w, h := frame.TextureOffset(j)
		SendPBOTextureToGPU(
			textureIDs[j], offset*3+j,
			w, h, channelType,
			frame.GLPixelBufferType,
			pboOffset, uint32(size),
		)
	}

	// map the buffer so that it can be written to
	buffer := gl.MapBuffer(frame.GLPixelBufferType, gl.WRITE_ONLY)
	if buffer == nil {
		panic("no buffer?")
	}
	frame.Data = unsafe.Slice((*byte)(buffer), frame.GLPixelBufferSize)
}

func GetFrameFromGPU(frame *encdec.Frame) {
	gl.BindBuffer(frame.GLPixelBufferType, frame.GLPixelBufferID)
	if frame.Data != nil {
		// since we're now going to write pixels into this frame
		// whoever was reading from it has released it and we can now
		// unmap its buffer
		gl.UnmapBuffer(frame.GLPixelBufferType)
	}

	gl.ReadPixels(0, 0, int32(frame.Width), int32(frame.Height), gl.RGB, gl.UNSIGNED_BYTE, gl.PtrOffset(0))

	// FIXME: Mapping the buffer immediately after calling ReadPixels is bad and forces it to be synchronous
	// however, even this is better than doing raw copy without PBO (but is this true on devices with integrated memory?)
	buffer := gl.MapBuffer(frame.GLPixelBufferType, gl.READ_ONLY)
	if buffer == nil {
		panic("no buffer?")
	}
	frame.Data = unsafe.Slice((*byte)(buffer), frame.GLPixelBufferSize)
}

type ThingWithFrames interface {
	Frames() *layer.FrameForwarder
}

func GetFrameFromGPUInto(into ThingWithFrames) {
	frames := into.Frames()
	frame := frames.GetFrameForWriting()
	if frame == nil {
		return // we are instructed to drop the frame
	}
	GetFrameFromGPU(frame)
	frames.FinishedWriting(frame)
}

func SendFramesToGPU[F ThingWithFrames](from []F, dt time.Duration) {
	isFirstFrame := (dt == 0)

	for i, thing := range from {
		frames := thing.Frames()

		frames.Age(dt)
		if frames.IsStill && !isFirstFrame {
			continue
		}

		frame := frames.GetFrameForReading()
		if frame == nil {
			continue // we are instructed to drop the frame
		}
		SendFrameToGPU(frame, frames.TextureIDs, int(i))
		frames.FinishedReading(frame)
	}
}
