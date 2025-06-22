package theatre

import (
	"fmt"

	"github.com/fosdem/fazantix/layer"
)

type Theatre struct {
	Layers []*layer.Layer
	Scenes map[string]*Scene
}

func New(sources []layer.Source, scenes map[string]*Scene, windowWidth int, windowHeight int) *Theatre {
	t := &Theatre{}

	t.Layers = make([]*layer.Layer, len(sources))

	for i, src := range sources {
		t.Layers[i] = layer.New(src, windowWidth, windowHeight)
	}

	t.Scenes = scenes

	return t
}

type Scene struct {
	Name        string
	LayerStates []*layer.LayerState
}

func (t *Theatre) NumLayers() int {
	return len(t.Layers)
}

func (t *Theatre) Start() {
	for _, l := range t.Layers {
		if l.Source.Start() {
			l.SetupTextures()
		}
	}
}

func (t *Theatre) Animate() {
	// todo: use delta-t
	for _, l := range t.Layers {
		l.Animate()
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
