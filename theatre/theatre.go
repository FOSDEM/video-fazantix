package theatre

import "github.com/fosdem/fazantix/layer"

type Scene struct {
}

type Theatre struct {
	Layers []*layer.Layer
	Scenes []*Scene
}

func (t *Theatre) NumLayers() int {
	return len(t.Layers)
}

func (t *Theatre) Start() {
	for _, l := range t.Layers {
		l.Source.Start()
		l.SetupTextures()
	}
}
