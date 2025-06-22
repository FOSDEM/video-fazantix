package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/pprof"

	"github.com/fosdem/fazantix/config"
	"github.com/fosdem/fazantix/theatre"
)

type Api struct {
	srv     http.Server
	mux     *http.ServeMux
	cfg     *config.ApiCfg
	theatre *theatre.Theatre
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
	if a.cfg.EnableProfiler {
		a.mux.Handle("/prof", pprof.Handler("goroutine"))
		a.mux.HandleFunc("/scene", a.handleScene)
	}

	return a.srv.ListenAndServe()
}

type SceneReq struct {
	Name string
}

func (a *Api) handleScene(w http.ResponseWriter, req *http.Request) {
	var sceneReq SceneReq
	err := json.NewDecoder(req.Body).Decode(&sceneReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not decode json request: %s", err), http.StatusBadRequest)
		return
	}

	err = a.theatre.SetScene(sceneReq.Name)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not set scene: %s", err), http.StatusForbidden)
		return
	}

	fmt.Fprintf(w, "\"ok\"\n")
}
