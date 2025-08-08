package rendering

import (
	"github.com/fosdem/fazantix/lib/layer"
	"github.com/go-gl/gl/v4.1-core/gl"
)

const f32 = 4

type GLVars struct {
	LayerPos      []float32
	LayerData     []float32
	StageData     uint32
	SourceIndices []uint32

	NumTextures int32
	NumLayers   int32
	NumSources  int32

	Program uint32

	// GL IDs
	VAO                  uint32
	VBO                  uint32
	Textures             []int32
	LayerDataUniform     int32
	LayerPosUniform      int32
	StageDataUniform     int32
	SourceIndicesUniform int32
	TexUniform           int32
}

func NewGLVars(program uint32, numSources int32, numLayers int32) *GLVars {
	g := &GLVars{}

	g.NumLayers = numLayers
	g.NumSources = numSources
	g.Program = program

	return g
}

func (g *GLVars) Start() {
	g.allocate()
	gl.ClearColor(1.0, 0.0, 0.0, 1.0)
	gl.UseProgram(g.Program)
	gl.Uniform1iv(g.TexUniform, g.NumTextures, &g.Textures[0])

}

func (g *GLVars) StartFrame() {
	gl.BindVertexArray(g.VAO)
}

func (g *GLVars) DrawStage(stage *layer.Stage) {
	frames := stage.Sink.Frames()

	gl.BindFramebuffer(gl.FRAMEBUFFER, frames.FramebufferID)

	// push vars related to the window stage
	g.loadStage(stage)
	g.pushStageVars()
}

func (g *GLVars) allocate() {

	// Configure the vertex data
	gl.GenVertexArrays(1, &g.VAO)
	gl.BindVertexArray(g.VAO)
	stride := int32(4 * f32)

	vertAttrib := uint32(gl.GetAttribLocation(g.Program, gl.Str("position\x00")))
	gl.EnableVertexAttribArray(vertAttrib)
	gl.VertexAttribPointerWithOffset(vertAttrib, 2, gl.FLOAT, false, stride, 0)

	texCoordAttrib := uint32(gl.GetAttribLocation(g.Program, gl.Str("uv\x00")))
	gl.EnableVertexAttribArray(texCoordAttrib)
	gl.VertexAttribPointerWithOffset(texCoordAttrib, 2, gl.FLOAT, false, stride, 2*f32)

	g.LayerPos = make([]float32, g.NumLayers*4)
	g.LayerPosUniform = gl.GetUniformLocation(g.Program, gl.Str("layerPosition\x00"))
	gl.Uniform4fv(g.LayerPosUniform, g.NumLayers, &g.LayerPos[0])

	g.LayerData = make([]float32, g.NumLayers*4)
	g.LayerDataUniform = gl.GetUniformLocation(g.Program, gl.Str("layerData\x00"))
	gl.Uniform4fv(g.LayerDataUniform, g.NumLayers, &g.LayerData[0])

	g.SourceIndicesUniform = gl.GetUniformLocation(g.Program, gl.Str("sourceIndices\x00"))
	gl.Uniform1uiv(g.SourceIndicesUniform, g.NumLayers, &g.SourceIndices[0])

	g.StageDataUniform = gl.GetUniformLocation(g.Program, gl.Str("stageData\x00"))
	gl.Uniform1ui(g.StageDataUniform, 0)

	// Allocate 3 textures for every source in case of planar YUV
	g.NumTextures = g.NumSources * 3
	g.Textures = make([]int32, g.NumTextures)
	for i := range g.NumTextures {
		g.Textures[i] = int32(i)
	}
	g.TexUniform = gl.GetUniformLocation(g.Program, gl.Str("tex\x00"))
	gl.Uniform1iv(g.TexUniform, g.NumTextures, &g.Textures[0])
}

func (g *GLVars) loadStage(stage *layer.Stage) {
	layers := stage.Layers
	for i := range g.NumLayers {
		g.LayerPos[(i*4)+0] = layers[i].Position.X
		g.LayerPos[(i*4)+1] = layers[i].Position.Y
		g.LayerPos[(i*4)+2] = layers[i].Size.X
		g.LayerPos[(i*4)+3] = layers[i].Size.Y
		g.LayerData[(i*4)+0] = layers[i].Opacity
		g.SourceIndices[i] = stage.SourceIndices[i]
	}
	g.StageData = stage.StageData()
}

func (g *GLVars) pushStageVars() {
	gl.Uniform1ui(g.StageDataUniform, g.StageData)
	gl.Uniform4fv(g.LayerDataUniform, g.NumLayers, &g.LayerData[0])
	gl.Uniform4fv(g.LayerPosUniform, g.NumLayers, &g.LayerPos[0])

	// draw vertices on the window stage
	gl.DrawArrays(gl.TRIANGLES, 0, 1*3)
}
