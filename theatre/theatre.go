package theatre

import (
	"fmt"

	"github.com/fosdem/fazantix/layer"
)

type Theatre struct {
	Layers []*layer.Layer
	Scenes map[string]*Scene
}

type Scene struct {
	LayerStates []*layer.LayerState
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

func (t *Theatre) SetScene(name string) error {
	if scene, ok := t.Scenes[name]; ok {
		for i, l := range t.Layers {
			l.ApplyState(scene.LayerStates[i])
		}
		return nil
	} else {
		return fmt.Errorf("no such scene: %s", name)
	}
}
