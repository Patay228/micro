package main

import (
	"log"
	"os"

	"circus-microservices/internal/common"
	"circus-microservices/internal/payment"

	"github.com/gin-gonic/gin"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8003"
	}

	broker := common.NewBroker("payment-service")
	defer broker.Close()

	var repo payment.Repository
	db, err := common.InitPostgresDB("payment", "migrations/000003_payment.up.sql")
	if err == nil {
		repo = payment.NewPostgresRepository(db)
		defer db.Close()
	} else {
		log.Printf("[payment-service] Using In-Memory Repository (%v)", err)
		repo = payment.NewInMemoryRepository()
	}

	r := gin.Default()
	handler := payment.NewHandler(repo, broker)
	handler.RegisterRoutes(r)

	log.Printf("[payment-service] Starting HTTP server on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to run payment service: %v", err)
	}
}
