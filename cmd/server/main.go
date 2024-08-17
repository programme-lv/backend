package main

import (
	"context"
	"log"

	"github.com/programme-lv/backend/http"
	"github.com/programme-lv/backend/subm"
)

func main() {
	submSrvc := subm.NewSubmissions(context.TODO())
	httpServer := http.NewHttpServer(submSrvc)

	address := ":8080"
	log.Printf("Starting server on %s", address)
	err := httpServer.Start(":8080")
	log.Printf("Server stopped with error: %v", err)
}
