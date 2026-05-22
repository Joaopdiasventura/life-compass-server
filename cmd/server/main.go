package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/Joaopdiasventura/life-compass-server/internal/database"
	"github.com/Joaopdiasventura/life-compass-server/internal/middleware"
	financeController "github.com/Joaopdiasventura/life-compass-server/internal/finance/controller"
	financeRepository "github.com/Joaopdiasventura/life-compass-server/internal/finance/repository"
	financeService "github.com/Joaopdiasventura/life-compass-server/internal/finance/service"
)

func main() {
	uri := os.Getenv("MONGODB_URI")
	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	databaseName := os.Getenv("MONGODB_DATABASE")
	if databaseName == "" {
		databaseName = "life_compass"
	}

	client := database.NewClient(uri)
	defer func() {
		if err := client.Disconnect(context.Background()); err != nil {
			log.Printf("Failed to disconnect MongoDB client: %v", err)
		}
	}()

	mongoDatabase := client.Database(databaseName)
	transactionRepository := financeRepository.NewMongoTransactionRepository(mongoDatabase)
	transactionService := financeService.NewTransactionService(transactionRepository)
	transactionController := financeController.NewTransactionController(transactionService)

	mux := http.NewServeMux()
	transactionController.RegisterRoutes(mux)

	server := &http.Server{
		Addr:    ":8080",
		Handler: middleware.CORS(mux, allowedOrigins),
	}

	log.Println("Server is running on :8080")

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
