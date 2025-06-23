package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/pprof"
	"time"

	"github.com/fosdem/fazantix/config"
	"github.com/fosdem/fazantix/rendering"
	"github.com/fosdem/fazantix/theatre"
)

type Api struct {
	srv          http.Server
	mux          *http.ServeMux
	cfg          *config.ApiCfg
	theatre      *theatre.Theatre
	start        time.Time
	FrameCounter uint64
}

func New(cfg *config.ApiCfg, theatre *theatre.Theatre) *Api {
	a := &Api{}
	a.cfg = cfg
	a.mux = http.NewServeMux()
	a.theatre = theatre
	a.srv.Addr = cfg.Bind
	a.srv.Handler = a.mux
	return a
}

func (a *Api) Serve() error {
	a.start = time.Now()
	if a.cfg.EnableProfiler {
		a.mux.HandleFunc("/prof", a.profileCPU)
	}
	a.mux.HandleFunc("/stats", a.stats)
	a.mux.HandleFunc("/scene", a.handleScene)
	a.mux.HandleFunc("/scene/{stage}/{scene}", a.handleScene)

	return a.srv.ListenAndServe()
}

type SceneReq struct {
	Scene string
	Stage string
}

func (a *Api) handleScene(w http.ResponseWriter, req *http.Request) {
	var sceneReq SceneReq
	if req.PathValue("scene") == "" && req.PathValue("stage") == "" {
		err := json.NewDecoder(req.Body).Decode(&sceneReq)
		if err != nil {
			http.Error(w, fmt.Sprintf("could not decode json request: %s", err), http.StatusBadRequest)
			return
		}
	} else {
		sceneReq.Scene = req.PathValue("scene")
		sceneReq.Stage = req.PathValue("stage")
	}

	err := a.theatre.SetScene(sceneReq.Stage, sceneReq.Scene)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not set scene: %s", err), http.StatusForbidden)
		return
	}

	fmt.Fprintf(w, "\"ok\"\n")
}

func (a *Api) profileCPU(w http.ResponseWriter, req *http.Request) {
	pprof.StartCPUProfile(w)
	time.Sleep(10 * time.Second)
	pprof.StopCPUProfile()
}

type Stats struct {
	TextureUpload      uint64  `json:"texture_upload"`
	TextureUploadAvgGb float64 `json:"texture_upload_avg_gb"`
	Uptime             float64 `json:"uptime"`
	TotalFrames        uint64  `json:"total_frames"`
	FPS                float64 `json:"fps"`
}

func (a *Api) stats(w http.ResponseWriter, req *http.Request) {
	uptime := float64(time.Since(a.start).Nanoseconds()) / 1e9
	stats := &Stats{
		Uptime:             uptime,
		TextureUpload:      rendering.TextureUploadCounter,
		TextureUploadAvgGb: float64(rendering.TextureUploadCounter) / (uptime * 1024 * 1024 * 1024),
		TotalFrames:        a.FrameCounter,
		FPS:                float64(a.FrameCounter) / uptime,
	}

	encoder := json.NewEncoder(w)
	err := encoder.Encode(stats)
	if err != nil {
		http.Error(w, fmt.Sprintf("could encode stats: %s", err), http.StatusForbidden)
		return
	}
	fmt.Fprintf(w, "\n")
}
