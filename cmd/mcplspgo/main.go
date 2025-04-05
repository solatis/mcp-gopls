package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hloiseau/mcplspgo/pkg/mcp"
)

func main() {
	server := mcp.NewServer()

	// MCP fonctionne sur STDIO, pas de besoin de port HTTP
	go func() {
		log.Println("MCP LSP Go server starting...")
		if err := server.ServeStdio(); err != nil {
			log.Fatalf("Error serving via STDIO: %v", err)
		}
	}()

	// Attendre le signal d'arrêt
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Server shutting down...")
	server.Close()
	log.Println("Server stopped")
}
