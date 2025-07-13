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
	"github.com/fosdem/fazantix/lib/rendering/shaders"
	"github.com/fosdem/fazantix/lib/v4lsource"
	"github.com/fosdem/fazantix/lib/windowsink"
)

type Listener func(theatre *Theatre, data interface{})
type Theatre struct {
	Sources            map[string]layer.Source
	SourceList         []layer.Source
	Scenes             map[string]*Scene
	Stages             map[string]*Stage
	WindowStageList    []*Stage
	NonWindowStageList []*Stage

	listener map[string][]Listener
}

type Stage struct {
	Layers       []*layer.Layer
	HFlip        bool
	VFlip        bool
	Sink         layer.Sink
	DefaultScene string
}

type EventDataSetScene struct {
	Event string
	Stage string
	Scene string
}

func New(cfg *config.Config, alloc encdec.FrameAllocator) (*Theatre, error) {
	sourceList, err := buildSourceList(cfg, alloc)
	if err != nil {
		return nil, err
	}
	sourceMap := buildSourceMap(sourceList)
	sceneMap := buildSceneMap(cfg, sourceList)
	stageMap := buildStageMap(cfg, sourceList, alloc)
	var windowStageList []*Stage
	var nonWindowStageList []*Stage

	for _, stage := range stageMap {
		switch stage.Sink.(type) {
		case *windowsink.WindowSink:
			windowStageList = append(windowStageList, stage)
		default:
			stage.HFlip = true
			nonWindowStageList = append(nonWindowStageList, stage)
		}
	}

	return &Theatre{
		Sources:            sourceMap,
		SourceList:         sourceList,
		Scenes:             sceneMap,
		Stages:             stageMap,
		WindowStageList:    windowStageList,
		NonWindowStageList: nonWindowStageList,
		listener:           make(map[string][]Listener),
	}, nil
}

func buildStageMap(cfg *config.Config, sources []layer.Source, alloc encdec.FrameAllocator) map[string]*Stage {
	stages := make(map[string]*Stage)
	for stageName, stageCfg := range cfg.Stages {
		stage := &Stage{}
		stage.Layers = make([]*layer.Layer, len(sources))
		stage.DefaultScene = stageCfg.DefaultScene

		for i, src := range sources {
			stage.Layers[i] = layer.New(src, stageCfg.Width, stageCfg.Height)
		}

		switch sc := stageCfg.SinkCfg.(type) {
		case *config.FFmpegSinkCfg:
			stage.Sink = ffmpegsink.New(stageName, sc, alloc)
		case *config.WindowSinkCfg:
			stage.Sink = windowsink.New(stageName, sc, alloc)
		default:
			panic(fmt.Sprintf("unhandled sink type: %+v", stageCfg.SinkCfg))
		}

		stages[stageName] = stage
	}
	return stages
}

func buildSceneMap(cfg *config.Config, sources []layer.Source) map[string]*Scene {
	scenes := make(map[string]*Scene)
	for sceneName, layerStateMap := range cfg.Scenes {
		layerStates := make([]*layer.LayerState, len(sources))
		for i, src := range sources {
			layerStates[i] = layerStateMap[src.Frames().Name]
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
	for _, layerStateMap := range cfg.Scenes {
		for name := range layerStateMap {
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
			src.Frames().SetupTextures()
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

func (t *Theatre) AddEventListener(event string, callback Listener) {
	t.listener[event] = append(t.listener[event], callback)
}

func (t *Theatre) invoke(event string, data interface{}) {
	for _, listener := range t.listener[event] {
		go listener(t, data)
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

func (t *Theatre) GetTheSingleWindowStage() *Stage {
	if len(t.WindowStageList) < 1 {
		panic("we still don't support running without a window-type sink :(")
	}
	if len(t.WindowStageList) > 1 {
		panic("we still don't support multiple window-type sinks :(")
	}
	return t.WindowStageList[0]
}

func (t *Theatre) ShaderData() *shaders.ShaderData {
	return &shaders.ShaderData{
		NumSources: t.NumSources(),
		Sources:    t.SourceList,
	}
}

func (s *Stage) StageData() uint32 {
	data := uint32(0)
	if s.HFlip {
		data += 1
	}
	if s.VFlip {
		data += 2
	}
	return data
}
