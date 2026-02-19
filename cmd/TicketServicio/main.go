// Package main es el punto de entrada del Ticket Daemon.
// Ticket Daemon es un servicio de Windows que recibe documentos JSON
// vía WebSocket y los imprime usando la librería Poster.
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/judwhite/go-svc"

	"github.com/adcondev/ticket-daemon/internal/daemon"
)

func main() {
	// Parse flags
	consoleMode := flag.Bool("console", false, "Run in console mode (not as service)")
	flag.Parse()

	prg := &daemon.Program{}

	// Check if running interactively (console mode)
	if *consoleMode || isInteractive() {
		runConsole(prg)
	} else {
		// Run as Windows Service
		if err := svc.Run(prg, syscall.SIGINT, syscall.SIGTERM); err != nil {
			log.Fatal(err)
		}
	}
}

// runConsole runs the program in console mode
func runConsole(prg *daemon.Program) {
	// Initialize
	if err := prg.Init(nil); err != nil {
		log.Fatalf("Init failed: %v", err)
	}

	// Start
	if err := prg.Start(); err != nil {
		log.Fatalf("Start failed: %v", err)
	}

	log.Println("═══════════════════════════════════════════════════════")
	log.Println("  🎫 TICKET SERVICIO - Modo Consola")
	log.Println("  Presiona Ctrl+C para detener...")
	log.Println("═══════════════════════════════════════════════════════")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("\n🛑 Shutting down...")
	err := prg.Stop()
	if err != nil {
		return
	}
}

// isInteractive checks if running from a terminal (not as service)
func isInteractive() bool {
	// Check if stdin is a terminal
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	// If stdin is a character device (terminal), we're interactive
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
