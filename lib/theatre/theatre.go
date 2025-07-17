package theatre

import (
	"fmt"
	"log"
	"sort"

	"github.com/fosdem/fazantix/lib/config"
	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/ffmpegsink"
	"github.com/fosdem/fazantix/lib/ffmpegsource"
	"github.com/fosdem/fazantix/lib/imgsource"
	"github.com/fosdem/fazantix/lib/layer"
	"github.com/fosdem/fazantix/lib/rendering"
	"github.com/fosdem/fazantix/lib/rendering/shaders"
	"github.com/fosdem/fazantix/lib/v4lsource"
	"github.com/fosdem/fazantix/lib/windowsink"
)

type Theatre struct {
	Sources    map[string]layer.Source
	SourceList []layer.Source
	Scenes     map[string]*Scene
	Stages     map[string]*layer.Stage

	WindowStageList    []*layer.Stage
	NonWindowStageList []*layer.Stage

	WindowSinkList []*windowsink.WindowSink

	ShutdownRequested bool

	listener map[string][]EventListener
}

func New(cfg *config.Config, alloc encdec.FrameAllocator) (*Theatre, error) {
	sourceList, err := buildSourceList(cfg, alloc)
	if err != nil {
		return nil, err
	}
	sourceMap := buildSourceMap(sourceList)
	sceneMap := buildSceneMap(cfg, sourceList)
	stageMap := buildStageMap(cfg, sourceList, alloc)
	var windowStageList []*layer.Stage
	var windowSinkList []*windowsink.WindowSink
	var nonWindowStageList []*layer.Stage

	for _, stage := range stageMap {
		switch sink := stage.Sink.(type) {
		case *windowsink.WindowSink:
			windowStageList = append(windowStageList, stage)
			windowSinkList = append(windowSinkList, sink)
		default:
			stage.HFlip = true
			nonWindowStageList = append(nonWindowStageList, stage)
		}
	}

	t := &Theatre{
		Sources:            sourceMap,
		SourceList:         sourceList,
		Scenes:             sceneMap,
		Stages:             stageMap,
		WindowStageList:    windowStageList,
		NonWindowStageList: nonWindowStageList,
		listener:           make(map[string][]EventListener),
		WindowSinkList:     windowSinkList,
	}

	err = t.ResetToDefaultScenes()
	if err != nil {
		return nil, err
	}

	return t, nil
}

func buildStageMap(cfg *config.Config, sources []layer.Source, alloc encdec.FrameAllocator) map[string]*layer.Stage {
	stages := make(map[string]*layer.Stage)
	for stageName, stageCfg := range cfg.Stages {
		stage := &layer.Stage{}
		stage.Layers = make([]*layer.Layer, len(sources))
		stage.DefaultScene = stageCfg.DefaultScene

		for i, src := range sources {
			stage.Layers[i] = layer.New(src, stageCfg.Width, stageCfg.Height)
		}

		switch sc := stageCfg.SinkCfg.(type) {
		case *config.FFmpegSinkCfg:
			stage.Sink = ffmpegsink.New(stageName, sc, &stageCfg.FrameCfg, alloc)
		case *config.WindowSinkCfg:
			stage.Sink = windowsink.New(stageName, sc, &stageCfg.FrameCfg, alloc)
		default:
			panic(fmt.Sprintf("unhandled sink type: %+v", stageCfg.SinkCfg))
		}

		stages[stageName] = stage
	}
	return stages
}

func buildDynamicScenes(cfg *config.Config) {
	for sourceName, source := range cfg.Sources {
		if source.MakeScene {
			scene := make(map[string]*config.LayerCfg)
			sourceLayer := &config.LayerCfg{
				LayerState: layer.LayerState{
					X:       0,
					Y:       0,
					Scale:   1,
					Opacity: 1,
				},
			}
			scene[sourceName] = sourceLayer
			cfg.Scenes[sourceName] = scene
		}
	}
}

func buildSceneMap(cfg *config.Config, sources []layer.Source) map[string]*Scene {
	buildDynamicScenes(cfg)
	scenes := make(map[string]*Scene)
	for sceneName, layerCfgMap := range cfg.Scenes {
		layerStates := make([]*layer.LayerState, len(sources))
		for i, src := range sources {
			layerStates[i] = layerCfgMap[src.Frames().Name].CopyState()
			log.Printf("layer state %d: %+v", i, layerStates[i])
		}
		scenes[sceneName] = &Scene{
			Name:        sceneName,
			LayerStates: layerStates,
		}
	}
	return scenes
}

func buildSourceList(cfg *config.Config, alloc encdec.FrameAllocator) ([]layer.Source, error) {
	enabledSources := make(map[string]struct{})
	for _, layerCfgMap := range cfg.Scenes {
		for name := range layerCfgMap {
			if _, ok := cfg.Sources[name]; ok {
				enabledSources[name] = struct{}{}
			} else {
				return nil, fmt.Errorf("no such source: %s", name)
			}
		}
	}

	var sortedSourceNames []string
	for name := range enabledSources {
		sortedSourceNames = append(sortedSourceNames, name)
	}

	sort.Slice(sortedSourceNames, func(i, j int) bool {
		ni := sortedSourceNames[i]
		nj := sortedSourceNames[j]
		return cfg.Sources[ni].Z < cfg.Sources[nj].Z
	})

	var sources []layer.Source
	for _, srcName := range sortedSourceNames {
		srcCfg := cfg.Sources[srcName]

		log.Printf("adding source: %s\n", srcName)

		switch sc := srcCfg.Cfg.(type) {
		case *config.FFmpegSourceCfg:
			sources = append(sources, ffmpegsource.New(srcName, sc, alloc))
		case *config.ImgSourceCfg:
			sources = append(sources, imgsource.New(srcName, sc, alloc))
		case *config.V4LSourceCfg:
			sources = append(sources, v4lsource.New(srcName, sc, alloc))
		default:
			panic(fmt.Sprintf("unhandled source type: %+v", srcCfg.Cfg))
		}
	}

	return sources, nil
}

func buildSourceMap(sources []layer.Source) map[string]layer.Source {
	sm := make(map[string]layer.Source)
	for _, src := range sources {
		sm[src.Frames().Name] = src
	}
	return sm
}

type Scene struct {
	Name        string
	LayerStates []*layer.LayerState
}

func (t *Theatre) NumLayers() int {
	return len(t.Sources) * len(t.Scenes)
}

func (t *Theatre) NumSources() int {
	return len(t.Sources)
}

func (t *Theatre) Start() {
	for _, stage := range t.WindowStageList {
		stage.Sink.Start()
	}
	for _, src := range t.Sources {
		if src.Start() {
			rendering.SetupTextures(src.Frames())
		}
	}
	for _, stage := range t.NonWindowStageList {
		stage.Sink.Start()
	}
}

func (t *Theatre) Animate(delta float32) {
	for _, s := range t.Stages {
		for _, l := range s.Layers {
			l.Animate(delta)
		}
	}
}

func (t *Theatre) SetScene(stageName string, sceneName string) error {
	if stage, ok := t.Stages[stageName]; ok {
		if scene, ok := t.Scenes[sceneName]; ok {
			t.invoke("set-scene", EventDataSetScene{
				Stage: stageName,
				Scene: sceneName,
			})
			for i, l := range stage.Layers {
				l.ApplyState(scene.LayerStates[i])
			}
			return nil
		} else {
			return fmt.Errorf("no such scene: %s", sceneName)
		}
	} else {
		return fmt.Errorf("no such stage: %s", stageName)
	}
}

func (t *Theatre) ResetToDefaultScenes() error {
	for name, stage := range t.Stages {
		err := t.SetScene(name, stage.DefaultScene)
		if err != nil {
			return fmt.Errorf(
				"could not apply default scene (%s) to stage %s: %w",
				stage.DefaultScene, name, err,
			)
		}
	}
	return nil
}

func (t *Theatre) ShaderData() *shaders.ShaderData {
	return &shaders.ShaderData{
		NumSources: t.NumSources(),
		Sources:    t.SourceList,
	}
}
