package rendering

import (
	"time"

	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/layer"
	"github.com/go-gl/gl/v4.1-core/gl"
)

func SendFrameToGPU(frame *encdec.Frame, textureIDs [3]uint32, offset int) {
	channelType := uint32(gl.RED)
	if frame.Type == encdec.RGBAFrames || frame.Type == encdec.YUV422pFrames {
		channelType = gl.RGBA
	}

	for j := 0; j < frame.NumTextures; j++ {
		dataPtr, w, h := frame.Texture(j)
		SendTextureToGPU(
			textureIDs[j], offset*3+j,
			w, h, channelType,
			dataPtr,
		)
	}
}

func GetFrameFromGPU(frame *encdec.Frame) {
	gl.ReadPixels(0, 0, int32(frame.Width), int32(frame.Height), gl.RGB, gl.UNSIGNED_BYTE, gl.Ptr(frame.Data))
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

	for i, thing := range from {
		frames := thing.Frames()

		frames.Age(dt)

		frame := frames.GetFrameForReading()
		if frame == nil {
			continue // we are instructed to drop the frame
		}
		SendFrameToGPU(frame, frames.TextureIDs, int(i))
		frames.FinishedReading(frame)
	}
}
