package impl

import (
	"context"
	"crypto/tls"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/luthermonson/go-proxmox"
	"github.com/rs/zerolog/log"
)

func Term(c *gin.Context) {
	insecureHTTPClient := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	credentials := proxmox.Credentials{
		Username: os.Getenv("PROXMOX_USERNAME"),
		Password: os.Getenv("PROXMOX_PASSWORD"),
	}
	client := proxmox.NewClient(os.Getenv("PROXMOX_URL"),
		proxmox.WithHTTPClient(&insecureHTTPClient),
		proxmox.WithCredentials(&credentials),
	)

	node, err := client.Node(context.Background(), os.Getenv("PROXMOX_NODE"))
	if err != nil {
		log.Error().Err(err).Msg("Error getting version")
		return
	}

	vmId, err := strconv.Atoi(os.Getenv("PROXMOX_VM"))
	if err != nil {
		log.Error().Err(err).Msg("Error getting version")
		return
	}

	vm, err := node.VirtualMachine(context.Background(), vmId)
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
	send <- "exit\n"
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
