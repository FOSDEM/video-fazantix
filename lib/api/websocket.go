package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(req *http.Request) bool {
		return true
	},
}

// @Summary	Open websocket for realtime status information
// @Router		/api/ws [get]
// @Param		Upgrade	header	string	true	"websocket"
// @Tags		base
// @Success	101
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
