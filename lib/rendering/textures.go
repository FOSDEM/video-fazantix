package rendering

import (
	_ "image/jpeg"
	_ "image/png"

	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/layer"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
)

func SetupTextures(f *layer.FrameForwarder) {
	width := f.Width
	height := f.Height

	switch f.FrameType {
	case encdec.YUV422Frames:
		f.TextureIDs[0] = SetupYUVTexture(width, height)
		f.TextureIDs[1] = SetupYUVTexture(width/2, height)
		f.TextureIDs[2] = SetupYUVTexture(width/2, height)
	case encdec.RGBAFrames:
		f.TextureIDs[0] = SetupRGBATexture(width, height, gl.RGBA)
	case encdec.BGRAFrames:
		f.TextureIDs[0] = SetupRGBATexture(width, height, gl.BGRA)
	case encdec.YUV422pFrames:
		f.TextureIDs[0] = SetupRGBATexture(width/2, height, gl.RGBA)
	case encdec.RGBFrames:
		f.TextureIDs[0] = SetupRGBTexture(width, height)
	default:
		panic("Unknown pixel format")
	}
}

func UseAsFramebuffer(f *layer.FrameForwarder) {
	if f.FrameType != encdec.RGBAFrames && f.FrameType != encdec.BGRAFrames {
		panic("trying to use a non-rgba frame forwarder as a framebuffer")
	}
	f.FramebufferID = UseTextureAsFramebuffer(f.TextureIDs[0])
}

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

func SetupRGBATexture(width int, height int, packing uint32) uint32 {
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
	buf := make([]uint8, width*height*4)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(width),
		int32(height),
		0,
		packing,
		gl.UNSIGNED_BYTE,
		gl.Ptr(&buf[0]),
	)
	return id
}

func SetupRGBTexture(width int, height int) uint32 {
	var id uint32
	gl.GenTextures(1, &id)
	gl.BindTexture(gl.TEXTURE_2D, id)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGB,
		int32(width),
		int32(height),
		0,
		gl.RGB,
		gl.UNSIGNED_BYTE,
		gl.Ptr(nil),
	)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	return id
}

func UseTextureAsFramebuffer(textureID uint32) uint32 {
	framebufferID := uint32(0)
	gl.GenFramebuffers(1, &framebufferID)
	gl.BindFramebuffer(gl.FRAMEBUFFER, framebufferID)

	gl.BindTexture(gl.TEXTURE_2D, textureID)
	gl.FramebufferTexture(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, textureID, 0)

	switch gl.CheckFramebufferStatus(gl.FRAMEBUFFER) {
	case gl.FRAMEBUFFER_COMPLETE:
	case gl.FRAMEBUFFER_INCOMPLETE_ATTACHMENT:
		panic("Framebuffer incomplete attachment")
	case gl.FRAMEBUFFER_INCOMPLETE_MISSING_ATTACHMENT:
		panic("FRAMEBUFFER_INCOMPLETE_MISSING_ATTACHMENT")
	case gl.FRAMEBUFFER_UNSUPPORTED:
		panic("FRAMEBUFFER_UNSUPPORTED")
	case gl.FRAMEBUFFER_INCOMPLETE_MULTISAMPLE:
		panic("FRAMEBUFFER_INCOMPLETE_MULTISAMPLE")
	default:
		panic("UNKNOWN FRAMEBUFFER ISSUE")
	}

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
