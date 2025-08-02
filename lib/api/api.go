package api

import (
	"embed"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io/fs"
	"log"
	"net/http"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/imgsource"
	"github.com/fosdem/fazantix/lib/layer"
	"github.com/gorilla/websocket"

	"github.com/fosdem/fazantix/lib/config"
	"github.com/fosdem/fazantix/lib/stats"
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
	a.mux.HandleFunc("/api/scene", a.handleScene)
	a.mux.HandleFunc("/api/scene/{stage}/{scene}", a.handleScene)
	a.mux.HandleFunc("/api/config", a.handleConfig)
	a.mux.HandleFunc("/api/ws", a.handleWebsocket)
	a.mux.HandleFunc("/api/media/source/{source}", a.handleMediaSource)
	a.mux.HandleFunc("/api/media/sink/{sink}", a.handleMediaSource)
	a.mux.HandleFunc("/api/media/source/{source}/{format}", a.handleMediaSource)
	a.mux.Handle("/", http.FileServer(http.FS(contentFS)))
	return a.srv.ListenAndServe()
}

type SceneReq struct {
	Scene string
	Stage string
}

type FrameForwarderObject interface {
	Frames() *layer.FrameForwarder
}

func (a *Api) handleMediaSource(w http.ResponseWriter, req *http.Request) {
	sourceName := req.PathValue("source")
	sinkName := req.PathValue("sink")
	formatName := req.PathValue("format")
	if sourceName == "" && sinkName == "" {
		http.Error(w, "Missing source name", http.StatusBadRequest)
		return
	}
	var source FrameForwarderObject
	if sourceName != "" {
		source = a.theatre.Sources[sourceName]
		if source == nil {
			http.Error(w, "Source does not exist", http.StatusNotFound)
			return
		}
	} else {
		stage := a.theatre.Stages[sinkName]
		if stage == nil {
			http.Error(w, "Sink does not exist", http.StatusNotFound)
			return
		}
		source = stage.Sink
	}

	if req.Method == "GET" {
		frame := source.Frames().GetAnyFrameForReading()
		if frame == nil {
			http.Error(w, "No frame returned", http.StatusFailedDependency)
			return
		}
		defer source.Frames().FinishedReading(frame)

		bounds := image.Rectangle{
			Min: image.Point{},
			Max: image.Point{X: frame.Width, Y: frame.Height},
		}

		var img image.Image
		switch frame.Type {
		case encdec.RGBAFrames:
			img = image.NewNRGBA(bounds)
			copy(img.(*image.NRGBA).Pix, frame.Data)
		case encdec.RGBFrames:
			img = image.NewNRGBA(bounds)
			pix := img.(*image.NRGBA).Pix
			for i := range len(frame.Data) / 3 {
				pix[i*4+0] = frame.Data[i*3+0]
				pix[i*4+1] = frame.Data[i*3+1]
				pix[i*4+2] = frame.Data[i*3+2]
				pix[i*4+3] = 255
			}
		case encdec.YUV422Frames:
			img = image.NewYCbCr(bounds, image.YCbCrSubsampleRatio422)
			textureY, _, _ := frame.Texture(0)
			textureCb, _, _ := frame.Texture(1)
			textureCr, _, _ := frame.Texture(2)
			copy(img.(*image.YCbCr).Y, textureY)
			copy(img.(*image.YCbCr).Cb, textureCb)
			copy(img.(*image.YCbCr).Cr, textureCr)
		default:
			http.Error(w, "Unhandled frame type", http.StatusInternalServerError)
			return
		}
		switch formatName {
		case "":
			fallthrough
		case "jpeg":
			err := jpeg.Encode(w, img, &jpeg.Options{Quality: 80})
			if err != nil {
				http.Error(w, "Could not jpeg encode this frame", http.StatusInternalServerError)
				return
			}
		case "png":
			err := png.Encode(w, img)
			if err != nil {
				http.Error(w, "Could not png encode this frame", http.StatusInternalServerError)
				return
			}
		default:
			http.Error(w, "Unsupported format", http.StatusBadRequest)
		}
		return
	}

	imgSource, ok := source.(*imgsource.ImgSource)
	if !ok {
		http.Error(w, "not a valid image source", http.StatusBadRequest)
		return
	}

	if req.Method == "PUT" {
		newImage, ftype, err := image.Decode(req.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("not a valid image: %s", err), http.StatusBadRequest)
			return
		}
		log.Printf("Image source %s was updated with new %s image (%dx%d)\n", sourceName, ftype, newImage.Bounds().Dx(), newImage.Bounds().Dy())
		err = imgSource.SetImage(newImage)
		if err != nil {
			http.Error(w, fmt.Sprintf("could not update image: %s", err), http.StatusBadRequest)
			return
		}
		return
	}

	http.Error(w, "Invalid method, only GET and PUT supported", http.StatusMethodNotAllowed)
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
		http.Error(w, fmt.Sprintf("could not set scene: %s", err), http.StatusBadRequest)
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

func (a *Api) suicide(w http.ResponseWriter, _ *http.Request) {
	log.Printf("shutting down as per api request")
	a.theatre.ShutdownRequested = true
	_, err := fmt.Fprintf(w, "\"ok\"\n")
	if err != nil {
		log.Printf("could not write response: %s\n", err.Error())
		return
	}
}

func (a *Api) getStats(w http.ResponseWriter, _ *http.Request) {
	encoder := json.NewEncoder(w)
	err := encoder.Encode(a.Stats)
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
	Stages []StageInfo `json:"stages"`
	Scenes []SceneInfo `json:"scenes"`
}
type StageInfo struct {
	Name       string
	PreviewFor string
}
type SceneInfo struct {
	Code  string
	Tag   string
	Label string
}

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

	a.Stats.WsClients = len(a.wsClients)

	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			delete(a.wsClients, ws)
			a.Stats.WsClients = len(a.wsClients)
			break
		}
		fmt.Printf("Received: %s\n", msg)
	}
}

func (a *Api) websocketWriter(ws *websocket.Conn) {

	for _, packet := range a.InitialState {
		if err := ws.WriteMessage(websocket.TextMessage, packet); err != nil {
			continue
		}
	}

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
		packet, err := json.Marshal(a.Stats)

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
