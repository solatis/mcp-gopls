package main

import (
	"fmt"
	"log"
	"os"

	"github.com/hloiseaufcms/MCPLSPGO/pkg/server"
)

func main() {
	service, err := server.NewService()
	if err != nil {
		fmt.Printf("Error creating service: %v", err)
		log.Fatalf("Error creating service: %v", err)
	}

	service.RegisterTools()

	if err := service.Start(); err != nil {
		fmt.Printf("Error running service: %v", err)
		log.Fatalf("Error running service: %v", err)
	}

	fmt.Println("Server shutdown normally")
	os.Exit(0)
}
