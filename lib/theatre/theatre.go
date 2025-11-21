package theatre

import (
	"fmt"
	"log"
	"time"

	"github.com/fosdem/fazantix/lib/config"
	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/layer"
	"github.com/fosdem/fazantix/lib/rendering"
	"github.com/fosdem/fazantix/lib/rendering/shaders"
	"github.com/fosdem/fazantix/lib/sink/ffmpegsink"
	"github.com/fosdem/fazantix/lib/sink/omtsink"
	"github.com/fosdem/fazantix/lib/sink/windowsink"
	"github.com/fosdem/fazantix/lib/source/ffmpegsource"
	"github.com/fosdem/fazantix/lib/source/htmlsource"
	"github.com/fosdem/fazantix/lib/source/imgsource"
	"github.com/fosdem/fazantix/lib/source/omtsource"
	"github.com/fosdem/fazantix/lib/source/v4lsource"
	"github.com/fosdem/fazantix/lib/utils"
)

type Theatre struct {
	SourceList      []layer.Source
	SourceIdxByName map[string]uint32
	Scenes          map[string]*Scene
	Stages          map[string]*layer.Stage

	FallbackSourceIndices []int32
	FallbackColour        utils.Colour

	WindowStageList    []*layer.Stage
	NonWindowStageList []*layer.Stage

	LayersPerStage uint32

	WindowSinkList []*windowsink.WindowSink

	ShutdownRequested bool

	listener map[string][]EventListener
}

func New(cfg *config.Config, alloc encdec.FrameAllocator, benchmark bool) (*Theatre, error) {
	buildDynamicScenes(cfg)
	sourceList, err := buildSourceList(cfg, alloc)
	if err != nil {
		return nil, err
	}
	sourceMap := buildSourceMap(sourceList)
	fallbackSourceIndices := buildFallbackSources(cfg, sourceMap)
	sceneMap := buildSceneMap(cfg, sourceList, sourceMap)
	stageMap, layersPerStage := buildStageMap(cfg, sourceList, sceneMap, alloc)
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
		SourceList:            sourceList,
		SourceIdxByName:       sourceMap,
		Scenes:                sceneMap,
		Stages:                stageMap,
		FallbackSourceIndices: fallbackSourceIndices,
		FallbackColour:        utils.ColourParse(cfg.FallbackColour),
		WindowStageList:       windowStageList,
		NonWindowStageList:    nonWindowStageList,
		listener:              make(map[string][]EventListener),
		WindowSinkList:        windowSinkList,
		LayersPerStage:        layersPerStage,
	}

	return t, nil
}

func buildStageMap(cfg *config.Config, sources []layer.Source, sceneMap map[string]*Scene, alloc encdec.FrameAllocator) (map[string]*layer.Stage, uint32) {
	layersPerSource := make([]uint32, len(sources))
	var layersPerStage uint32
	for _, scene := range sceneMap {
		for i := range sources {
			cnt := uint32(len(scene.LayerStatesBySourceIdx[i]))
			if layersPerSource[i] < cnt {
				layersPerSource[i] = cnt
			}
		}
	}
	for _, n := range layersPerSource {
		layersPerStage += n
	}

	stages := make(map[string]*layer.Stage)
	for stageName, stageCfg := range cfg.Stages {
		stage := &layer.Stage{}
		stage.SetSpeed(time.Duration(*stageCfg.TransitionTimeMs) * time.Millisecond)
		stage.Layers = make([]*layer.Layer, len(sources))
		stage.LayersByScene = make(map[string][]*layer.Layer)
		stage.SourceIndices = make([]int32, layersPerStage)
		stage.SourceTypes = make([]encdec.FrameType, len(sources))
		stage.DefaultScene = stageCfg.DefaultScene
		stage.PreviewFor = stageCfg.StageCfgStub.PreviewFor
		stage.RateDivisor = stageCfg.StageCfgStub.Rate.RateDivisor
		stage.RateOffset = stageCfg.StageCfgStub.Rate.RateOffset

		// create a distinct layer collection for each stage
		layersBySource := make([][]*layer.Layer, len(sources))
		for i, src := range sources {
			layersBySource[i] = make([]*layer.Layer, layersPerSource[i])
			for j := range layersPerSource[i] {
				layersBySource[i][j] = layer.New(uint32(i), src, stageCfg.Width, stageCfg.Height)
			}
			stage.SourceTypes[i] = src.Frames().FrameType
		}

		for sceneName, scene := range sceneMap {
			layerIndices := make([]uint32, len(sources))
			// SourceOrder may have repeating elements
			for _, srcIdx := range scene.SourceOrder {
				stage.LayersByScene[sceneName] = append(
					stage.LayersByScene[sceneName],
					layersBySource[srcIdx][layerIndices[srcIdx]],
				)
				layerIndices[srcIdx] += 1
			}
			// add placeholders for unused values
			for srcIdx := range sources {
				for layerIndices[srcIdx] < layersPerSource[srcIdx] {
					stage.LayersByScene[sceneName] = append(
						stage.LayersByScene[sceneName],
						layersBySource[srcIdx][layerIndices[srcIdx]],
					)
					layerIndices[srcIdx] += 1
				}
			}

			if len(stage.LayersByScene[sceneName]) != int(layersPerStage) {
				panic(fmt.Sprintf(
					"bad layer count for stage %s and scene %s: %d against %d",
					stageName, sceneName,
					len(stage.LayersByScene[sceneName]), int(layersPerStage),
				))
			}
		}

		switch sc := stageCfg.SinkCfg.(type) {
		case *config.FFmpegSinkCfg:
			stage.Sink = ffmpegsink.New(stageName, sc, &stageCfg.FrameCfg, alloc)
		case *config.WindowSinkCfg:
			stage.Sink = windowsink.New(stageName, sc, &stageCfg.FrameCfg, alloc)
		case *config.OmtSinkCfg:
			stage.Sink = omtsink.New(stageName, sc, &stageCfg.FrameCfg, alloc)
		default:
			panic(fmt.Sprintf("unhandled sink type: %+v", stageCfg.SinkCfg))
		}

		stages[stageName] = stage
	}
	return stages, layersPerStage
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

func buildSceneMap(cfg *config.Config, sources []layer.Source, sourceIdxByName map[string]uint32) map[string]*Scene {
	scenes := make(map[string]*Scene)

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
			srcIdx := sourceIdxByName[layerCfg.SourceName]
			scene.LayerStatesBySourceIdx[srcIdx] = append(
				scene.LayerStatesBySourceIdx[srcIdx],
				layerCfg.CopyState(),
			)
			scene.SourceOrder = append(scene.SourceOrder, srcIdx)
		}
		scenes[sceneName] = scene
	}
	return scenes
}

func addEnabledSource(srcName string, cfg *config.Config, enabledSources map[string]struct{}) error {
	if _, ok := cfg.Sources[srcName]; !ok {
		return fmt.Errorf("no such source: %s", srcName)
	}

	enabledSources[srcName] = struct{}{}

	if cfg.Sources[srcName].Fallback != "" {
		err := addEnabledSource(
			cfg.Sources[srcName].Fallback,
			cfg, enabledSources,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func buildSourceList(cfg *config.Config, alloc encdec.FrameAllocator) ([]layer.Source, error) {
	enabledSources := make(map[string]struct{})
	for _, sceneCfg := range cfg.Scenes {
		for _, layerCfg := range sceneCfg.Layers {
			err := addEnabledSource(layerCfg.SourceName, cfg, enabledSources)
			if err != nil {
				return nil, err
			}
		}
	}

	var sources []layer.Source
	for srcName := range enabledSources {
		srcCfg := cfg.Sources[srcName]

		switch sc := srcCfg.Cfg.(type) {
		case *config.FFmpegSourceCfg:
			sources = append(sources, ffmpegsource.New(srcName, sc, alloc))
		case *config.ImgSourceCfg:
			sources = append(sources, imgsource.New(srcName, sc, alloc))
		case *config.V4LSourceCfg:
			sources = append(sources, v4lsource.New(srcName, sc))
		case *config.HtmlSourceCfg:
			sources = append(sources, htmlsource.New(srcName, sc, alloc))
		case *config.OmtSourceCfg:
			sources = append(sources, omtsource.New(srcName, sc, alloc))
		default:
			panic(fmt.Sprintf("unhandled source type: %+v", srcCfg.Cfg))
		}
	}

	return sources, nil
}

func buildFallbackSources(cfg *config.Config, sourceIdxByName map[string]uint32) []int32 {
	fallbackSources := make([]int32, len(sourceIdxByName))
	for name, idx := range sourceIdxByName {
		srcCfg := cfg.Sources[name]
		if srcCfg.Fallback != "" {
			fallbackSources[idx] = int32(sourceIdxByName[srcCfg.Fallback])
		} else {
			fallbackSources[idx] = -1
		}
	}

	return fallbackSources
}

func buildSourceMap(sources []layer.Source) map[string]uint32 {
	sm := make(map[string]uint32)
	for i, src := range sources {
		sm[src.Frames().Name] = uint32(i)
	}
	return sm
}

type Scene struct {
	Name                   string
	Tag                    string
	Label                  string
	LayerStatesBySourceIdx [][]*layer.LayerState
	SourceOrder            []uint32
}

func (t *Theatre) NumSources() int {
	return len(t.SourceList)
}

func (t *Theatre) Start() {
	err := t.ResetToDefaultScenes()
	if err != nil {
		return
	}
	refreshRate := int(0)
	for _, stage := range t.WindowStageList {
		stage.Sink.Start()
		refreshRate = stage.Sink.(*windowsink.WindowSink).GetRefreshRate()
	}
	for _, src := range t.SourceList {
		if src.Start() {
			rendering.SetupTextures(src.Frames())
		}
	}

	for _, stage := range t.NonWindowStageList {
		if stage.RateDivisor < 1 {
			stage.RateDivisor = 1
		}
		stage.Sink.SetRate(refreshRate / int(stage.RateDivisor))
		if stage.RateDivisor > 1 {
			log.Printf("setting sink rate to %d", refreshRate/int(stage.RateDivisor))
		}

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
	idxBySrc := make([]int, len(t.SourceList))

	if stage, ok := t.Stages[stageName]; ok {
		if scene, ok := t.Scenes[sceneName]; ok {
			t.invoke("set-scene", EventDataSetScene{
				Stage: stageName,
				Scene: sceneName,
			})

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
				stage.SourceIndices[i] = int32(layer.SourceIdx)
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
		NumSources:     uint32(t.NumSources()),
		Sources:        t.SourceList,
		NumLayers:      t.LayersPerStage,
		FallbackColour: t.FallbackColour,
	}
}

func (t *Theatre) SourceByName(name string) layer.Source {
	idx, ok := t.SourceIdxByName[name]
	if !ok {
		return nil
	}
	return t.SourceList[idx]
}
