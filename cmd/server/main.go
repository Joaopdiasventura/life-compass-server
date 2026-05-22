package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Joaopdiasventura/life-compass-server/internal/database"
)

func main() {
	uri := os.Getenv("MONGODB_URI")

	database.NewClient(uri)

	mux := http.NewServeMux()

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("Server is running on :8080")

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
