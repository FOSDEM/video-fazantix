package rendering

import (
	"time"

	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/layer"
	"github.com/go-gl/gl/v4.1-core/gl"
)

func SendFrameToGPU(frame *encdec.Frame, textureIDs [3]uint32, offset int) {
	channelType := uint32(gl.RED)
	if frame.Type == encdec.RGBAFrames {
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

type ThingWithFrames interface {
	Frames() *layer.FrameForwarder
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
			continue
		}
		SendFrameToGPU(frame, frames.TextureIDs, int(i))
		frames.FinishedReading(frame)
	}
}
