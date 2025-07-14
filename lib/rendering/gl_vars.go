package rendering

import (
	"github.com/fosdem/fazantix/lib/layer"
	"github.com/go-gl/gl/v4.1-core/gl"
)

const f32 = 4

type GLVars struct {
	LayerPos  []float32
	LayerData []float32
	StageData uint32

	NumTextures int32
	NumLayers   int32

	// GL IDs
	VAO              uint32
	VBO              uint32
	Textures         []int32
	LayerDataUniform int32
	LayerPosUniform  int32
	StageDataUniform int32
	TexUniform       int32
}

func AllocateGLVars(program uint32, numLayers int32) *GLVars {
	g := &GLVars{}

	g.NumLayers = numLayers

	vertices := []float32{
		//  X, Y,  U, V
		-1.0, -1.0, 0.0, 1.0,
		+1.0, -1.0, 1.0, 1.0,
		+1.0, +1.0, 1.0, 0.0,

		-1.0, -1.0, 0.0, 1.0,
		+1.0, +1.0, 1.0, 0.0,
		-1.0, +1.0, 0.0, 0.0,
	}

	// Configure the vertex data
	gl.GenVertexArrays(1, &g.VAO)
	gl.BindVertexArray(g.VAO)

	gl.GenBuffers(1, &g.VBO)
	gl.BindBuffer(gl.ARRAY_BUFFER, g.VBO)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*f32, gl.Ptr(vertices), gl.STATIC_DRAW)

	stride := int32(4 * f32)

	vertAttrib := uint32(gl.GetAttribLocation(program, gl.Str("position\x00")))
	gl.EnableVertexAttribArray(vertAttrib)
	gl.VertexAttribPointerWithOffset(vertAttrib, 2, gl.FLOAT, false, stride, 0)

	texCoordAttrib := uint32(gl.GetAttribLocation(program, gl.Str("uv\x00")))
	gl.EnableVertexAttribArray(texCoordAttrib)
	gl.VertexAttribPointerWithOffset(texCoordAttrib, 2, gl.FLOAT, false, stride, 2*f32)

	g.LayerPos = make([]float32, numLayers*4)
	g.LayerPosUniform = gl.GetUniformLocation(program, gl.Str("layerPosition\x00"))
	gl.Uniform4fv(g.LayerPosUniform, numLayers, &g.LayerPos[0])

	g.LayerData = make([]float32, numLayers*4)
	g.LayerDataUniform = gl.GetUniformLocation(program, gl.Str("layerData\x00"))
	gl.Uniform4fv(g.LayerDataUniform, numLayers, &g.LayerData[0])

	g.StageDataUniform = gl.GetUniformLocation(program, gl.Str("stageData\x00"))
	gl.Uniform1ui(g.StageDataUniform, 0)

	// Allocate 3 textures for every layer in case of planar YUV
	g.NumTextures = numLayers * 3
	g.Textures = make([]int32, g.NumTextures)
	for i := range g.NumTextures {
		g.Textures[i] = int32(i)
	}
	g.TexUniform = gl.GetUniformLocation(program, gl.Str("tex\x00"))
	gl.Uniform1iv(g.TexUniform, g.NumTextures, &g.Textures[0])

	return g
}

func (g *GLVars) PushCommonVars() {
	gl.Uniform1iv(g.TexUniform, g.NumTextures, &g.Textures[0])
}

func (g *GLVars) ReadLayers(layers []*layer.Layer) {
	for i := range g.NumLayers {
		g.LayerPos[(i*4)+0] = layers[i].Position.X
		g.LayerPos[(i*4)+1] = layers[i].Position.Y
		g.LayerPos[(i*4)+2] = layers[i].Size.X
		g.LayerPos[(i*4)+3] = layers[i].Size.Y
		g.LayerData[(i*4)+0] = layers[i].Opacity
	}
}

func (g *GLVars) PushStageVars() {
	gl.Uniform1ui(g.StageDataUniform, g.StageData)
	gl.Uniform4fv(g.LayerDataUniform, g.NumLayers, &g.LayerData[0])
	gl.Uniform4fv(g.LayerPosUniform, g.NumLayers, &g.LayerPos[0])

	// draw vertices on the window stage
	gl.DrawArrays(gl.TRIANGLES, 0, 2*3)
}

func (g *GLVars) DrawStage(stage *layer.Stage) {
	frames := stage.Sink.Frames()

	gl.BindFramebuffer(gl.FRAMEBUFFER, frames.FramebufferID)
	gl.Viewport(0, 0, int32(frames.Width), int32(frames.Height))
	gl.Clear(gl.COLOR_BUFFER_BIT)

	// push vars related to the window stage
	g.ReadLayers(stage.Layers)
	g.StageData = stage.StageData()
	g.PushStageVars()
}
