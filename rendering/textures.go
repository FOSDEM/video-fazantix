package rendering

import (
	_ "image/jpeg"
	_ "image/png"

	"github.com/fosdem/fazantix/encdec"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

func SetupYUVTexture(width int, height int) uint32 {
	var id uint32
	gl.GenTextures(1, &id)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, id)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

	// this is to compenasate for floating-point errors on x==0/y==0
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)

	buf := make([]uint8, width*height)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RED,
		int32(width),
		int32(height),
		0,
		gl.RED,
		gl.UNSIGNED_BYTE,
		gl.Ptr(&buf[0]),
	)
	return id
}

func SetupRGBTexture(width int, height int, texture []byte) uint32 {
	var id uint32
	gl.GenTextures(1, &id)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, id)
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
		gl.Ptr(texture),
	)
	return id
}

func UseTextureAsFramebuffer(textureID uint32) uint32 {
	gl.FramebufferTexture(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, textureID, 0)

	if gl.CheckFramebufferStatus(gl.FRAMEBUFFER) != gl.FRAMEBUFFER_COMPLETE {
		panic("Framebuffer failed")
	}

	framebufferID := uint32(0)
	gl.GenFramebuffers(1, &framebufferID)
	gl.BindFramebuffer(gl.FRAMEBUFFER, framebufferID)
	return framebufferID
}

var TextureUploadCounter uint64

func SendTextureToGPU(texID uint32, offset int, w int, h int, channelType uint32, data []byte) {
	gl.ActiveTexture(uint32(gl.TEXTURE0 + offset))
	gl.BindTexture(gl.TEXTURE_2D, texID)
	gl.TexSubImage2D(
		gl.TEXTURE_2D,
		0, 0, 0,
		int32(w), int32(h),
		channelType, gl.UNSIGNED_BYTE, gl.Ptr(data),
	)
	TextureUploadCounter += uint64(len(data))
}

func SendFrameToGPU(frame *encdec.ImageData, textureIDs [3]uint32, offset int) {
	channelType := uint32(gl.RED)
	if frame.Type == encdec.RGBFrames {
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
