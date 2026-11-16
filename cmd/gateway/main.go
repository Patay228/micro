package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"circus-microservices/internal/common"

	"github.com/gin-gonic/gin"
)

const swaggerHTML = `<!DOCTYPE html>
<html lang="ru">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Цирковая ИС — Swagger UI (API Gateway)</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.18.2/swagger-ui.css" />
  <link rel="icon" type="image/png" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.18.2/favicon-32x32.png" sizes="32x32" />
  <style>
    html { box-sizing: border-box; overflow-y: scroll; }
    *, *:before, *:after { box-sizing: inherit; }
    body { margin: 0; background: #fafafa; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; }
    .topbar { background-color: #1e1b4b !important; padding: 12px 0; box-shadow: 0 2px 8px rgba(0,0,0,0.15); }
    .topbar-wrapper .link { display: flex; align-items: center; color: #fff; font-weight: bold; font-size: 1.2rem; text-decoration: none; }
    .topbar-wrapper .link span { color: #f59e0b; margin-left: 10px; font-weight: normal; font-size: 0.95rem; }
    .swagger-ui .info { margin: 25px 0; }
    .swagger-ui .info .title { font-size: 2rem; color: #1e1b4b; }
    .swagger-ui .scheme-container { background: #fff; box-shadow: 0 1px 3px rgba(0,0,0,0.08); margin: 0 0 20px 0; padding: 15px 0; }
  </style>
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.18.2/swagger-ui-bundle.js" charset="UTF-8"></script>
<script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.18.2/swagger-ui-standalone-preset.js" charset="UTF-8"></script>
<script>
window.onload = function() {
  const ui = SwaggerUIBundle({
    urls: [
      { url: "/swagger-specs/catalog-service.yaml", name: "1. Catalog Service (Афиша, Шоу, Сеансы, Схемы залов)" },
      { url: "/swagger-specs/booking-service.yaml", name: "2. Booking Service (Бронирование мест, Билеты, Сага)" },
      { url: "/swagger-specs/payment-service.yaml", name: "3. Payment Service (Платежи, Банковский вебхук)" },
      { url: "/swagger-specs/loyalty-service.yaml", name: "4. Loyalty Service (Программа лояльности, Кэшбэк)" },
      { url: "/swagger-specs/notification-service.yaml", name: "5. Notification Service (SMS и Email оповещения)" }
    ],
    "urls.primaryName": "1. Catalog Service (Афиша, Шоу, Сеансы, Схемы залов)",
    dom_id: '#swagger-ui',
    deepLinking: true,
    presets: [
      SwaggerUIBundle.presets.apis,
      SwaggerUIStandalonePreset
    ],
    plugins: [
      SwaggerUIBundle.plugins.DownloadUrl
    ],
    layout: "StandaloneLayout",
    persistAuthorization: true,
    displayRequestDuration: true
  });
  window.ui = ui;
};
</script>
</body>
</html>`

func findAPIDir() string {
	candidates := []string{"./api", "../api", "../../api"}
	for _, p := range candidates {
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			return p
		}
	}
	return "./api"
}

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

	// Swagger UI и OpenAPI спецификации
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger")
	})
	r.GET("/docs", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger")
	})
	r.GET("/swagger", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerHTML))
	})
	r.GET("/swagger-specs/:filename", func(c *gin.Context) {
		filename := filepath.Base(c.Param("filename"))
		apiDir := findAPIDir()
		filePath := filepath.Join(apiDir, filename)
		data, err := os.ReadFile(filePath)
		if err != nil {
			common.RenderProblem(c, http.StatusNotFound, "Not Found", "OpenAPI spec not found: "+filename, c.Request.URL.Path)
			return
		}
		c.Data(http.StatusOK, "application/yaml; charset=utf-8", data)
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
	log.Printf("[gateway-service] Swagger UI documentation available at: http://localhost:%s/swagger", port)
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

