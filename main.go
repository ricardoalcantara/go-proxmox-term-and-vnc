package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/ricardoalcantara/go-proxmox-term-and-vnc/impl"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
		return filepath.Base(file) + ":" + strconv.Itoa(line)
	}
	log.Logger = log.
		With().
		Caller().
		Logger().
		Output(zerolog.ConsoleWriter{Out: os.Stderr})
	godotenv.Load()

	log.Info().Msg("starting server")
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "hello world")
	})
	r.GET("/term", impl.Term)
	r.GET("/vnc", impl.Vnc)

	r.RunTLS(":8523", "server.crt", "server.key")
}
