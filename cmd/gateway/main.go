package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"circus-microservices/internal/common"

	"github.com/gin-gonic/gin"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	catalogURL, _ := url.Parse(getEnv("CATALOG_URL", "http://localhost:8001"))
	bookingURL, _ := url.Parse(getEnv("BOOKING_URL", "http://localhost:8002"))
	paymentURL, _ := url.Parse(getEnv("PAYMENT_URL", "http://localhost:8003"))
	loyaltyURL, _ := url.Parse(getEnv("LOYALTY_URL", "http://localhost:8004"))
	notificationURL, _ := url.Parse(getEnv("NOTIFICATION_URL", "http://localhost:8005"))

	proxyCatalog := httputil.NewSingleHostReverseProxy(catalogURL)
	proxyBooking := httputil.NewSingleHostReverseProxy(bookingURL)
	proxyPayment := httputil.NewSingleHostReverseProxy(paymentURL)
	proxyLoyalty := httputil.NewSingleHostReverseProxy(loyaltyURL)
	proxyNotification := httputil.NewSingleHostReverseProxy(notificationURL)

	r := gin.Default()

	// Защита: запрет прямого доступа к внутренним эндпоинтам /internal/*
	r.Use(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/internal") {
			common.RenderProblem(c, http.StatusForbidden, "Forbidden", "Access to internal endpoints is blocked from API Gateway", c.Request.URL.Path)
			c.Abort()
			return
		}
		c.Next()
	})

	// Маршрутизация на микросервисы
	r.Any("/api/v1/shows/*action", func(c *gin.Context) { proxyCatalog.ServeHTTP(c.Writer, c.Request) })
	r.Any("/api/v1/shows", func(c *gin.Context) { proxyCatalog.ServeHTTP(c.Writer, c.Request) })
	r.Any("/api/v1/sessions/*action", func(c *gin.Context) { proxyCatalog.ServeHTTP(c.Writer, c.Request) })
	r.Any("/api/v1/sessions", func(c *gin.Context) { proxyCatalog.ServeHTTP(c.Writer, c.Request) })

	r.Any("/api/v1/bookings/*action", func(c *gin.Context) { proxyBooking.ServeHTTP(c.Writer, c.Request) })
	r.Any("/api/v1/bookings", func(c *gin.Context) { proxyBooking.ServeHTTP(c.Writer, c.Request) })

	r.Any("/api/v1/payments/*action", func(c *gin.Context) { proxyPayment.ServeHTTP(c.Writer, c.Request) })
	r.Any("/api/v1/payments", func(c *gin.Context) { proxyPayment.ServeHTTP(c.Writer, c.Request) })

	r.Any("/api/v1/loyalty/*action", func(c *gin.Context) { proxyLoyalty.ServeHTTP(c.Writer, c.Request) })
	r.Any("/api/v1/loyalty", func(c *gin.Context) { proxyLoyalty.ServeHTTP(c.Writer, c.Request) })

	r.Any("/api/v1/notifications/*action", func(c *gin.Context) { proxyNotification.ServeHTTP(c.Writer, c.Request) })
	r.Any("/api/v1/notifications", func(c *gin.Context) { proxyNotification.ServeHTTP(c.Writer, c.Request) })

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":   "UP",
			"services": []string{"catalog", "booking", "payment", "loyalty", "notification"},
		})
	})

	log.Printf("[gateway-service] API Gateway running on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to run gateway: %v", err)
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
