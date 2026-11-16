package common

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

// InitPostgresDB пытается подключиться к PostgreSQL по указанному URL (или DATABASE_URL).
// Если успешно, применяет SQL-скрипт миграции.
func InitPostgresDB(serviceName string, migrationFile string) (*sql.DB, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv(strings.ToUpper(serviceName) + "_DATABASE_URL")
	}
	if dbURL == "" {
		return nil, fmt.Errorf("no DATABASE_URL provided for %s", serviceName)
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping failed for %s: %w", serviceName, err)
	}

	log.Printf("[%s] Successfully connected to PostgreSQL: %s", serviceName, dbURL)

	// Применение миграции, если файл существует
	if migrationFile != "" {
		content, err := os.ReadFile(migrationFile)
		if err == nil {
			_, err = db.Exec(string(content))
			if err != nil {
				log.Printf("[%s] Warning applying migration %s: %v", serviceName, migrationFile, err)
			} else {
				log.Printf("[%s] Applied migration from %s", serviceName, migrationFile)
			}
		}
	}

	return db, nil
}
