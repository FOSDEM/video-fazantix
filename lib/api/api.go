package api

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"maps"
	"net/http"
	"runtime/pprof"
	"slices"
	"time"

	"github.com/gorilla/websocket"

	"github.com/fosdem/fazantix/lib/config"
	"github.com/fosdem/fazantix/lib/rendering"
	"github.com/fosdem/fazantix/lib/theatre"
)

//go:embed static/*
var content embed.FS
var contentFS, _ = fs.Sub(content, "static")

type Api struct {
	srv     http.Server
	mux     *http.ServeMux
	cfg     *config.ApiCfg
	theatre *theatre.Theatre
	start   time.Time
	FPS     int

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

	t.AddEventListener("set-scene", func(t *theatre.Theatre, data interface{}) {
		event := data.(theatre.EventDataSetScene)
		event.Event = "set-scene"
		log.Printf("Scene switched on stage %s to scene %s\n", event.Stage, event.Scene)

		for ws := range a.wsClients {
			packet, err := json.Marshal(event)
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
	return a
}

func (a *Api) Serve() error {
	a.start = time.Now()
	if a.cfg.EnableProfiler {
		a.mux.HandleFunc("/prof", a.profileCPU)
	}
	a.mux.HandleFunc("/api/kill", a.suicide)
	a.mux.HandleFunc("/api/stats", a.stats)
	a.mux.HandleFunc("/api/scene", a.handleScene)
	a.mux.HandleFunc("/api/scene/{stage}/{scene}", a.handleScene)
	a.mux.HandleFunc("/api/config", a.handleConfig)
	a.mux.HandleFunc("/api/ws", a.handleWebsocket)
	a.mux.Handle("/", http.FileServer(http.FS(contentFS)))
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

	_, err = fmt.Fprintf(w, "\"ok\"\n")
	if err != nil {
		log.Printf("could not write response: %s\n", err.Error())
		return
	}
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

type Stats struct {
	TextureUpload      uint64  `json:"texture_upload"`
	TextureUploadAvgGb float64 `json:"texture_upload_avg_gb"`
	Uptime             float64 `json:"uptime"`
	FPS                int     `json:"fps"`
	WsClients          int     `json:"ws_clients"`
}

func (a *Api) suicide(w http.ResponseWriter, _ *http.Request) {
	log.Printf("shutting down as per api request")
	a.theatre.ShutdownRequested = true
	_, err := fmt.Fprintf(w, "\"ok\"\n")
	if err != nil {
		log.Printf("could not write response: %s\n", err.Error())
		return
	}
}

func (a *Api) stats(w http.ResponseWriter, _ *http.Request) {
	uptime := float64(time.Since(a.start).Nanoseconds()) / 1e9
	stats := &Stats{
		Uptime:             uptime,
		TextureUpload:      rendering.TextureUploadCounter,
		TextureUploadAvgGb: float64(rendering.TextureUploadCounter) / (uptime * 1024 * 1024 * 1024),
		FPS:                a.FPS,
		WsClients:          len(a.wsClients),
	}

	encoder := json.NewEncoder(w)
	err := encoder.Encode(stats)
	if err != nil {
		http.Error(w, fmt.Sprintf("could encode stats: %s", err), http.StatusForbidden)
		return
	}
	_, err = fmt.Fprintf(w, "\n")
	if err != nil {
		log.Printf("could not write response: %s\n", err.Error())
		return
	}
}

type Config struct {
	Stages []string `json:"stages"`
	Scenes []string `json:"scenes"`
}

func (a *Api) handleConfig(w http.ResponseWriter, _ *http.Request) {
	result := &Config{
		Stages: slices.Collect(maps.Keys(a.theatre.Stages)),
		Scenes: slices.Collect(maps.Keys(a.theatre.Scenes)),
	}
	encoder := json.NewEncoder(w)
	err := encoder.Encode(result)
	if err != nil {
		http.Error(w, fmt.Sprintf("couldn't encode config: %s", err), http.StatusForbidden)
		return
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(req *http.Request) bool {
		return true
	},
}

func (a *Api) handleWebsocket(w http.ResponseWriter, req *http.Request) {
	ws, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("couldn't make websocket: %s", err), 400)
		return
	}
	defer func(ws *websocket.Conn) {
		err := ws.Close()
		if err != nil {
			log.Printf("could not close websocket: %s\n", err.Error())
		}
	}(ws)
	a.wsClients[ws] = true

	go a.websocketWriter(ws)

	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			delete(a.wsClients, ws)
			break
		}
		fmt.Printf("Received: %s\n", msg)
	}
}

func (a *Api) websocketWriter(ws *websocket.Conn) {
	pingTicker := time.NewTicker(2 * time.Second)
	defer func() {
		pingTicker.Stop()
		err := ws.Close()
		if err != nil {
			log.Printf("could not close websocket: %s\n", err.Error())
			return
		}
	}()
	timeout := 10 * time.Second
	for range pingTicker.C {
		uptime := float64(time.Since(a.start).Nanoseconds()) / 1e9
		stats := &Stats{
			Uptime:             uptime,
			TextureUpload:      rendering.TextureUploadCounter,
			TextureUploadAvgGb: float64(rendering.TextureUploadCounter) / (uptime * 1024 * 1024 * 1024),
			FPS:                a.FPS,
			WsClients:          len(a.wsClients),
		}
		packet, err := json.Marshal(stats)
		if err != nil {
			return
		}
		err = ws.SetWriteDeadline(time.Now().Add(timeout))
		if err != nil {
			log.Printf("could not set write deadline: %s\n", err.Error())
			return
		}
		if err := ws.WriteMessage(websocket.TextMessage, packet); err != nil {
			return
		}
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
