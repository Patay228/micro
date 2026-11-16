package main

import (
	"log"
	"os"

	"circus-microservices/internal/booking"
	"circus-microservices/internal/common"

	"github.com/gin-gonic/gin"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8002"
	}

	broker := common.NewBroker("booking-service")
	defer broker.Close()

	var repo booking.Repository
	db, err := common.InitPostgresDB("booking", "migrations/000002_booking.up.sql")
	if err == nil {
		repo = booking.NewPostgresRepository(db)
		defer db.Close()
	} else {
		log.Printf("[booking-service] Using In-Memory Repository (%v)", err)
		repo = booking.NewInMemoryRepository()
	}

	saga := booking.NewSagaCoordinator(repo, broker)
	handler := booking.NewHandler(repo, saga, broker)

	r := gin.Default()
	handler.RegisterRoutes(r)

	log.Printf("[booking-service] Starting HTTP server on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to run booking service: %v", err)
	}
}
