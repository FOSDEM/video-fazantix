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
