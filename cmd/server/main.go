package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/Joaopdiasventura/life-compass-server/internal/database"
	financeControllers "github.com/Joaopdiasventura/life-compass-server/internal/finance/controller"
	financeRepositories "github.com/Joaopdiasventura/life-compass-server/internal/finance/repository"
	financeServices "github.com/Joaopdiasventura/life-compass-server/internal/finance/service"
	"github.com/Joaopdiasventura/life-compass-server/internal/middleware"
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
	transactionRepository := financeRepositories.NewMongoTransactionRepository(mongoDatabase)
	goalRepository := financeRepositories.NewMongoGoalRepository(mongoDatabase)
	financeService := financeServices.NewFinanceService(financeServices.FinanceServiceDependencies{
		TransactionRepository: transactionRepository,
		GoalRepository:        goalRepository,
	})
	financeController := financeControllers.NewFinanceController(financeService)

	mux := http.NewServeMux()
	financeController.RegisterRoutes(mux)

	server := &http.Server{
		Addr:    ":8080",
		Handler: middleware.CORS(mux, allowedOrigins),
	}

	log.Println("Server is running on :8080")

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
