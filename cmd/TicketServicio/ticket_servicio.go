// Package main es el punto de entrada del Ticket Daemon.
// Ticket Daemon es un servicio de Windows que recibe documentos JSON
// vía WebSocket y los imprime usando la librería Poster.
package main

import (
	"log"
	"syscall"

	"github.com/judwhite/go-svc"

	"github.com/adcondev/ticket-daemon/internal/daemon"
)

func main() {
	prg := &daemon.Program{}

	if err := svc.Run(prg, syscall.SIGINT, syscall.SIGTERM); err != nil {
		log.Fatal(err)
	}
}
