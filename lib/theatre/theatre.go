package theatre

import (
	"fmt"
	"sort"
	"time"

	"github.com/fosdem/fazantix/lib/config"
	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/layer"
	"github.com/fosdem/fazantix/lib/rendering"
	"github.com/fosdem/fazantix/lib/rendering/shaders"
	"github.com/fosdem/fazantix/lib/sink/ffmpegsink"
	"github.com/fosdem/fazantix/lib/sink/windowsink"
	"github.com/fosdem/fazantix/lib/source/ffmpegsource"
	"github.com/fosdem/fazantix/lib/source/imgsource"
	"github.com/fosdem/fazantix/lib/source/v4lsource"
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
	buildDynamicScenes(cfg)
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

	return t, nil
}

func buildStageMap(cfg *config.Config, sources []layer.Source, alloc encdec.FrameAllocator) map[string]*layer.Stage {
	stages := make(map[string]*layer.Stage)
	for stageName, stageCfg := range cfg.Stages {
		stage := &layer.Stage{}
		stage.SetSpeed(time.Duration(*stageCfg.TransitionTimeMs) * time.Millisecond)
		stage.Layers = make([]*layer.Layer, len(sources))
		stage.DefaultScene = stageCfg.DefaultScene
		stage.PreviewFor = stageCfg.StageCfgStub.PreviewFor

		for i, src := range sources {
			stage.Layers[i] = layer.New(uint32(i), src, stageCfg.Width, stageCfg.Height)

			panic("for each source they may be multiple layers, if the source appears multiple times in some scene")
			// here, build stage.LayersByScene and stage.SourceIndicesByScene
			// use a new z-order determined by the order in each scene instead of global
			// then, set stage.Layers and stage.SourceIndices in ApplyScene()
			panic("need to compute stage.SourceIndices")
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
			cfg.Scenes[sourceName] = &config.SceneCfg{
				Tag:   source.Tag,
				Label: source.Label,
				Layers: []*config.LayerCfg{
					{
						SourceName: sourceName,
						Transform: &config.LayerTransformCfg{
							LayerTransform: layer.LayerTransform{
								X:       0,
								Y:       0, // if I put a comment here this code will look pointy
								Scale:   1,
								Opacity: 1,
							},
						},
					},
				},
			}
		}
	}
}

func buildSceneMap(cfg *config.Config, sources []layer.Source) map[string]*Scene {
	scenes := make(map[string]*Scene)

	sourceIndexByName := make(map[string]int)
	for i := range sources {
		sourceIndexByName[sources[i].Frames().Name] = i
	}

	for sceneName, sceneCfg := range cfg.Scenes {
		if sceneCfg.Label == "" {
			sceneCfg.Label = sceneName
		}
		if sceneCfg.Tag == "" {
			sceneCfg.Tag = sceneName[0:3] + sceneName[len(sceneName)-1:]
		}
		scene := &Scene{
			Name:                   sceneName,
			Label:                  sceneCfg.Label,
			Tag:                    sceneCfg.Tag,
			LayerStatesBySourceIdx: make([][]*layer.LayerState, len(sources)),
		}

		for _, layerCfg := range sceneCfg.Layers {
			if layerCfg.SourceName != "" {
				srcIdx := sourceIndexByName[layerCfg.SourceName]
				scene.LayerStatesBySourceIdx[srcIdx] = append(
					scene.LayerStatesBySourceIdx[srcIdx],
					layerCfg.CopyState(),
				)
			} else if layerCfg.StageName != "" {
				panic("stage sources not yet supported")
			}
		}
		scenes[sceneName] = scene
	}
	return scenes
}

func buildSourceList(cfg *config.Config, alloc encdec.FrameAllocator) ([]layer.Source, error) {
	enabledSources := make(map[string]struct{})
	for _, sceneCfg := range cfg.Scenes {
		for _, layerCfg := range sceneCfg.Layers {
			if _, ok := cfg.Sources[layerCfg.SourceName]; ok {
				enabledSources[layerCfg.SourceName] = struct{}{}
			} else {
				return nil, fmt.Errorf("no such source: %s", layerCfg.SourceName)
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

		switch sc := srcCfg.Cfg.(type) {
		case *config.FFmpegSourceCfg:
			sources = append(sources, ffmpegsource.New(srcName, sc, alloc))
		case *config.ImgSourceCfg:
			sources = append(sources, imgsource.New(srcName, sc, alloc))
		case *config.V4LSourceCfg:
			sources = append(sources, v4lsource.New(srcName, sc))
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
	Name                   string
	Tag                    string
	Label                  string
	LayerStatesBySourceIdx [][]*layer.LayerState
}

func (t *Theatre) NumLayers() int {
	panic("this is incorrect")
	return len(t.Sources) * len(t.Scenes)
}

func (t *Theatre) NumSources() int {
	return len(t.Sources)
}

func (t *Theatre) Start() {
	err := t.ResetToDefaultScenes()
	if err != nil {
		return
	}

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
			l.Animate(delta, s.Speed)
		}
	}
}

func (t *Theatre) SetTransitionSpeed(stageName string, transitionDuration time.Duration) error {
	if stage, ok := t.Stages[stageName]; ok {
		stage.SetSpeed(transitionDuration)
		return nil
	} else {
		return fmt.Errorf("no such stage: %s", stageName)
	}
}

func (t *Theatre) SetScene(stageName string, sceneName string, transition bool) error {
	idxBySrc := make([]int, len(t.Sources))

	if stage, ok := t.Stages[stageName]; ok {
		if scene, ok := t.Scenes[sceneName]; ok {
			stage.Layers = stage.LayersByScene[sceneName]
			for i, layer := range stage.Layers {
				j := idxBySrc[layer.SourceIdx]
				idxBySrc[layer.SourceIdx] += 1
				layerStatesForThisSource := scene.LayerStatesBySourceIdx[layer.SourceIdx]
				if j < len(layerStatesForThisSource) {
					layer.ApplyState(layerStatesForThisSource[j], transition)
				} else {
					// make the rest of the layers for this source invisible
					layer.ApplyState(nil, false)
				}
				stage.SourceIndices[i] = layer.SourceIdx
			}
		} else {
			return fmt.Errorf("no such stage: %s", stageName)
		}
	} else {
		return fmt.Errorf("no such stage: %s", stageName)
	}

	return nil
}

func (t *Theatre) ResetToDefaultScenes() error {
	for name, stage := range t.Stages {
		err := t.SetScene(name, stage.DefaultScene, false)
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
