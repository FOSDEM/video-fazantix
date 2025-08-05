package api

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/fosdem/fazantix/lib/config"
	"github.com/fosdem/fazantix/lib/stats"
	"github.com/fosdem/fazantix/lib/theatre"

	_ "github.com/fosdem/fazantix/docs"
	httpSwagger "github.com/swaggo/http-swagger"
)

//	@title			Fazantix API
//	@version		1.0
//	@description	This is the control API for the Fazantix vision mixer

//go:embed static/*
var content embed.FS
var contentFS, _ = fs.Sub(content, "static")

type Api struct {
	srv     http.Server
	mux     *http.ServeMux
	cfg     *config.ApiCfg
	theatre *theatre.Theatre

	Stats *stats.Stats

	InitialState map[string][]byte
	stateMutex   sync.Mutex

	wsClients map[*websocket.Conn]bool
}

func New(cfg *config.ApiCfg, t *theatre.Theatre) *Api {
	a := &Api{}
	a.cfg = cfg
	a.mux = http.NewServeMux()
	a.theatre = t
	a.srv.Addr = cfg.Bind
	a.srv.Handler = a.mux
	a.wsClients = make(map[*websocket.Conn]bool)
	a.InitialState = make(map[string][]byte)

	t.AddEventListener("set-scene", func(t *theatre.Theatre, data interface{}) {
		a.stateMutex.Lock()
		defer a.stateMutex.Unlock()
		event := data.(theatre.EventDataSetScene)
		event.Event = "set-scene"
		log.Printf("Scene switched on stage %s to scene %s\n", event.Stage, event.Scene)
		packet, err := json.Marshal(event)
		a.InitialState[fmt.Sprintf("active-scene-%s", event.Stage)] = packet

		for ws := range a.wsClients {
			if err != nil {
				return
			}
			err = ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err != nil {
				log.Printf("could not set write deadline: %s\n", err.Error())
				return
			}
			if err := ws.WriteMessage(websocket.TextMessage, packet); err != nil {
				return
			}
		}
	})
	a.Stats = stats.New()
	return a
}

func (a *Api) Serve() error {
	if a.cfg.EnableProfiler {
		a.mux.HandleFunc("/prof", a.profileCPU)
	}
	a.mux.HandleFunc("/api/kill", a.suicide)
	a.mux.HandleFunc("/api/stats", a.getStats)
	a.mux.HandleFunc("/api/scene", a.handleSceneJson)
	a.mux.HandleFunc("/api/scene/{stage}/{scene}", a.handleScene)
	a.mux.HandleFunc("/api/config", a.handleConfig)
	a.mux.HandleFunc("/api/ws", a.handleWebsocket)
	a.mux.HandleFunc("/api/media/source/{source}", a.handleMediaSource)
	a.mux.HandleFunc("/api/media/sink/{sink}", a.handleMediaSource)
	a.mux.HandleFunc("/api/media/source/{source}/{format}", a.handleMediaSource)
	a.mux.Handle("/swagger/", httpSwagger.Handler())
	a.mux.Handle("/", http.FileServer(http.FS(contentFS)))
	return a.srv.ListenAndServe()
}

func (a *Api) profileCPU(w http.ResponseWriter, _ *http.Request) {
	err := pprof.StartCPUProfile(w)
	if err != nil {
		http.Error(w, fmt.Sprintf("Could not start CPU profile: %s", err), http.StatusInternalServerError)
		return
	}
	time.Sleep(10 * time.Second)
	pprof.StopCPUProfile()
}

// @Summary	Shut down Fazantix
// @Router		/api/kill [post]
// @Tags		base
// @Success	200
// @Success	200	{object}	stats.Stats
func (a *Api) suicide(w http.ResponseWriter, _ *http.Request) {
	log.Printf("shutting down as per api request")
	a.theatre.ShutdownRequested = true
	_, err := fmt.Fprintf(w, "\"ok\"\n")
	if err != nil {
		log.Printf("could not write response: %s\n", err.Error())
		return
	}
}

// @Summary	Get internal runtime statistics
// @Router		/api/stats [get]
// @Tags		base
// @Accept		json
// @Produce	json
// @Success	200	{object}	stats.Stats
func (a *Api) getStats(w http.ResponseWriter, _ *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	err := encoder.Encode(a.Stats)
	if err != nil {
		http.Error(w, fmt.Sprintf("could encode stats: %s", err), http.StatusInternalServerError)
		return
	}
	_, err = fmt.Fprintf(w, "\n")
	if err != nil {
		log.Printf("could not write response: %s\n", err.Error())
		return
	}
}

type Config struct {
	Stages []StageInfo `json:"stages"`
	Scenes []SceneInfo `json:"scenes"`
}
type StageInfo struct {
	Name       string `example:"projector"`
	PreviewFor string
}
type SceneInfo struct {
	Code  string `example:"side-by-side"`
	Tag   string `example:"SbS"`
	Label string `example:"Side by side"`
}

// @Summary	Get list of stages and scenes
// @Router		/api/config [get]
// @Tags		base
// @Accept		json
// @Produce	json
// @Success	200	{object}	api.Config
func (a *Api) handleConfig(w http.ResponseWriter, _ *http.Request) {
	result := &Config{
		Stages: make([]StageInfo, len(a.theatre.Stages)),
		Scenes: make([]SceneInfo, len(a.theatre.Scenes)),
	}
	idx := 0
	for name, scene := range a.theatre.Scenes {
		result.Scenes[idx].Code = name
		result.Scenes[idx].Label = scene.Label
		result.Scenes[idx].Tag = scene.Tag
		idx++
	}
	idx = 0
	for name, stage := range a.theatre.Stages {
		result.Stages[idx].Name = name
		result.Stages[idx].PreviewFor = stage.PreviewFor
		idx++
	}
	w.Header().Add("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	err := encoder.Encode(result)
	if err != nil {
		http.Error(w, fmt.Sprintf("couldn't encode config: %s", err), http.StatusForbidden)
		return
	}
}

func ServeInBackground(theatre *theatre.Theatre, cfg *config.ApiCfg) *Api {
	var theApi *Api
	if cfg != nil {
		theApi = New(cfg, theatre)

		log.Printf("starting web server\n")
		go func() {
			err := theApi.Serve()
			if err != nil {
				log.Fatalf("could not start web server: %s", err)
			}
		}()
	}
	return theApi
}
