package main

import (
	"log"
	"os"

	"circus-microservices/internal/catalog"
	"circus-microservices/internal/common"

	"github.com/gin-gonic/gin"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8001"
	}

	broker := common.NewBroker("catalog-service")
	defer broker.Close()

	// Инициализация репозитория (Postgres или InMemory)
	var repo catalog.Repository
	db, err := common.InitPostgresDB("catalog", "migrations/000001_catalog.up.sql")
	if err == nil {
		repo = catalog.NewPostgresRepository(db)
		defer db.Close()
	} else {
		log.Printf("[catalog-service] Using In-Memory Repository (%v)", err)
		repo = catalog.NewInMemoryRepository()
	}

	r := gin.Default()
	handler := catalog.NewHandler(repo, broker)
	handler.RegisterRoutes(r)

	log.Printf("[catalog-service] Starting HTTP server on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to run catalog service: %v", err)
	}
}
