package main

import (
	"context"
	"log"

	"aicon-coding-test/internal/infrastructure/server"
)

func main() {
	ctx := context.Background()

	server := server.NewServer()

	if err := server.Run(ctx); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
