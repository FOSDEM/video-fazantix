package rendering

import (
	"github.com/fosdem/fazantix/lib/layer"
	"github.com/fosdem/fazantix/lib/utils"
	"github.com/go-gl/gl/v4.1-core/gl"
)

const f32 = 4

type GLVars struct {
	LayerPos      []float32
	LayerData     []float32
	StageData     uint32
	SourceIndices []int32
	SourceTypes   []uint32

	NumTextures int32
	NumLayers   int32
	Sources     []layer.Source

	// FallbackIndices stores an index for each source which acts as
	// a fallback source, or -1 if such does not exist
	FallbackIndices []int32

	Program uint32

	BGColour utils.Colour

	// GL IDs
	VAO                  uint32
	VBO                  uint32
	Textures             []int32
	LayerDataUniform     int32
	LayerPosUniform      int32
	StageDataUniform     int32
	SourceIndicesUniform int32
	SourceTypesUniform   int32
	TexUniform           int32
}

func NewGLVars(program uint32, numLayers int32, sources []layer.Source, fallbackSourceIndices []int32, bgColour utils.Colour) *GLVars {
	g := &GLVars{}

	g.NumLayers = numLayers
	g.Sources = sources
	g.FallbackIndices = fallbackSourceIndices
	g.Program = program
	g.BGColour = bgColour

	return g
}

func (g *GLVars) Start() {
	g.allocate()
	gl.ClearColor(g.BGColour.R, g.BGColour.G, g.BGColour.B, g.BGColour.A)
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

	g.SourceIndices = make([]int32, g.NumLayers)
	g.SourceIndicesUniform = gl.GetUniformLocation(g.Program, gl.Str("sourceIndices\x00"))
	gl.Uniform1iv(g.SourceIndicesUniform, g.NumLayers, &g.SourceIndices[0])

	g.SourceTypes = make([]uint32, len(g.Sources))
	g.SourceTypesUniform = gl.GetUniformLocation(g.Program, gl.Str("sourceTypes\x00"))
	gl.Uniform1uiv(g.SourceTypesUniform, int32(len(g.Sources)), &g.SourceTypes[0])

	g.StageDataUniform = gl.GetUniformLocation(g.Program, gl.Str("stageData\x00"))
	gl.Uniform1ui(g.StageDataUniform, 0)

	// Allocate 3 textures for every source in case of planar YUV
	g.NumTextures = int32(len(g.Sources) * 3)
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

		sourceIndex := stage.SourceIndices[i]
		for sourceIndex != -1 && !g.Sources[sourceIndex].Frames().IsReady {
			sourceIndex = g.FallbackIndices[sourceIndex]
		}
		g.SourceIndices[i] = sourceIndex
	}
	for i := range len(g.Sources) {
		g.SourceTypes[i] = uint32(stage.SourceTypes[i])
	}
	g.StageData = stage.StageData()
}

func (g *GLVars) pushStageVars() {
	gl.Uniform1ui(g.StageDataUniform, g.StageData)
	gl.Uniform4fv(g.LayerDataUniform, g.NumLayers, &g.LayerData[0])
	gl.Uniform4fv(g.LayerPosUniform, g.NumLayers, &g.LayerPos[0])
	gl.Uniform1iv(g.SourceIndicesUniform, g.NumLayers, &g.SourceIndices[0])
	gl.Uniform1uiv(g.SourceTypesUniform, int32(len(g.Sources)), &g.SourceTypes[0])

	// draw vertices on the window stage
	gl.DrawArrays(gl.TRIANGLES, 0, 1*3)
}
