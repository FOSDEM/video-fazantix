package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type SceneReq struct {
	Stage string `example:"projector"`
	Scene string `example:"side-by-side"`
}

// @Summary	Start a transition to a specific scene on one of the outputs
// @Router		/api/scene/{stage}/{scene} [post]
// @Tags		scene
// @Param		stage	path	string	true	"Output name to switch the scene for"
// @Param		scene	path	string	true	"The name of the scene to transition to"
// @Success	200
// @Failure	400	{string}	string	"Could not decode json request"
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

// @Summary	Start a transition to a specific scene on one of the outputs
// @Router		/api/scene [post]
// @Param		sceneReq	body	SceneReq	true	"Transition"
// @Tags		scene
// @Accept		json
// @Produce	json
// @Success	200
// @Failure	400	{string}	string	"Could not decode json request"
func (a *Api) handleSceneJson(w http.ResponseWriter, req *http.Request) {
	var sceneReq SceneReq
	err := json.NewDecoder(req.Body).Decode(&sceneReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not decode json request: %s", err), http.StatusBadRequest)
		return
	}

	err = a.theatre.SetScene(sceneReq.Stage, sceneReq.Scene)
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
