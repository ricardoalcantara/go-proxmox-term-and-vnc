package impl

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

func Term(c *gin.Context) {
	vm, err := GetVm()
	if err != nil {
		log.Error().Err(err).Msg("Error getting version")
		return
	}

	vnc, err := vm.TermProxy(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("Error getting version")
		return
	}

	send, recv, errs, close, err := vm.VNCWebSocket(vnc)
	if err != nil {
		log.Error().Err(err).Msg("Error getting version")
		return
	}
	defer close()

	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	done := make(chan struct{})
	go reader(ws, send, done)
	go writer(ws, recv, errs, done)

	<-done
}

func reader(ws *websocket.Conn, send chan string, done chan struct{}) {
	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error().Err(err).Send()
			}
			done <- struct{}{}
			return
		}
		send <- string(msg)
	}
}

func writer(ws *websocket.Conn, recv chan string, errs chan error, done chan struct{}) {
	for {
		select {
		case msg := <-recv:
			if msg != "" {
				err := ws.WriteMessage(websocket.TextMessage, []byte(msg))
				if err != nil {
					done <- struct{}{}
					log.Error().Err(err).Send()
					return
				}
			}

		case err := <-errs:
			if err != nil {
				log.Error().Err(err).Send()
			}
			done <- struct{}{}
			return
		case <-done:
			return
		}

	}
}
