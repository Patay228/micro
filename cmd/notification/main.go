package main

import (
	"log"
	"os"

	"circus-microservices/internal/common"
	"circus-microservices/internal/notification"

	"github.com/gin-gonic/gin"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8005"
	}

	broker := common.NewBroker("notification-service")
	defer broker.Close()

	var repo notification.Repository
	db, err := common.InitPostgresDB("notification", "migrations/000005_notification.up.sql")
	if err == nil {
		repo = notification.NewPostgresRepository(db)
		defer db.Close()
	} else {
		log.Printf("[notification-service] Using In-Memory Repository (%v)", err)
		repo = notification.NewInMemoryRepository()
	}

	r := gin.Default()
	handler := notification.NewHandler(repo, broker)
	handler.RegisterRoutes(r)

	log.Printf("[notification-service] Starting HTTP server on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to run notification service: %v", err)
	}
}
