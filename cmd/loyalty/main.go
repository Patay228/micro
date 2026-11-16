package main

import (
	"log"
	"os"

	"circus-microservices/internal/common"
	"circus-microservices/internal/loyalty"

	"github.com/gin-gonic/gin"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8004"
	}

	broker := common.NewBroker("loyalty-service")
	defer broker.Close()

	var repo loyalty.Repository
	db, err := common.InitPostgresDB("loyalty", "migrations/000004_loyalty.up.sql")
	if err == nil {
		repo = loyalty.NewPostgresRepository(db)
		defer db.Close()
	} else {
		log.Printf("[loyalty-service] Using In-Memory Repository (%v)", err)
		repo = loyalty.NewInMemoryRepository()
	}

	r := gin.Default()
	handler := loyalty.NewHandler(repo, broker)
	handler.RegisterRoutes(r)

	log.Printf("[loyalty-service] Starting HTTP server on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to run loyalty service: %v", err)
	}
}
