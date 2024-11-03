package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"encrypted-db/config"
	"encrypted-db/docs"
	"encrypted-db/internal/db"
	"encrypted-db/internal/handlers/admin"
	"encrypted-db/internal/handlers/public"
	"encrypted-db/internal/handlers/socket"
	"encrypted-db/internal/handlers/system"
	"encrypted-db/internal/rabbitmq"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Encrypted-DB API Documentation
// @version 1.0
// @description This is a sample server for the encrypted-db project.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /
// @schemes http https

// @x-warning.CORS "Cross-Origin Resource Sharing (CORS) error occurs when trying to access the API from an unauthorized domain. Make sure the origin domain is allowed in CORS settings on the server."
// @x-warning.NetworkFailure "Network Failure error may happen if there is an issue with the network connection while making a request. Check your internet connection and try again."
// @x-warning.URLScheme "The URL scheme error 'URL scheme must be \"http\" or \"https\" for CORS request' occurs when the URL protocol is not http or https. Ensure the URL scheme is correctly set to http or https."
func main() {
	config.LoadConfig()

	// Initialize database connections
	postgresService := db.NewPostgresService()
	defer postgresService.Close() // Ensures PostgreSQL connection is closed when main exits

	redisService := db.NewRedisService()
	defer redisService.Close() // Ensures Redis connection is closed when main exits

	// Setup RabbitMQ service
	rabbitMQService := rabbitmq.NewRabbitMQService()
	defer rabbitMQService.Close() // Ensures RabbitMQ connection is closed when main exits

	// Initialize WebSocket handler with RabbitMQ
	webSocketHandler := socket.NewWebSocketHandler(rabbitMQService) // Using the new socket package

	systemHandler := system.NewHandler(postgresService, redisService, rabbitMQService)
	publicHandler := public.NewHandler(postgresService, redisService, rabbitMQService)
	adminHandler := admin.NewHandler(postgresService, redisService, rabbitMQService)

	// Load and cache currencies on server start
	err := adminHandler.LoadAndCacheCurrencies()
	if err != nil {
		log.Fatalf("Failed to load and cache currencies: %v", err)
	}

	serverAddr := fmt.Sprintf("%s:%s", config.Config.Server.IP, config.Config.Server.Port)
	swaggerURL := fmt.Sprintf("http://%s/swagger/index.html", serverAddr)

	// Set up gin and Swagger documentation
	r := gin.Default()
	docs.SwaggerInfo.BasePath = "/"
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler)) // Swagger endpoint

	// Root endpoint
	r.GET("/", func(c *gin.Context) {
		c.String(200, "Welcome to the Encrypted-DB API")
	})

	// WebSocket route for currencies
	socket := r.Group("/ws")
	{
		socket.GET("/", webSocketHandler.ServeWSGin) // WebSocket endpoint for currency updates
	}

	// System routes
	system := r.Group("/system")
	{
		system.GET("/healthcheck", systemHandler.HealthCheckHandler)
		system.GET("/ping", systemHandler.PingPongHandler)
	}

	// Public routes
	public := r.Group("/public")
	{
		public.GET("/currencies", publicHandler.GetActiveCurrencies)
		public.GET("/currencies/:hk", publicHandler.GetCurrencyByHK)
	}

	// Admin routes
	admin := r.Group("/admin")
	{
		admin.POST("/currencies", adminHandler.CreateCurrency)
		admin.PUT("/currencies/:hk", adminHandler.UpdateCurrency)
		admin.DELETE("/currencies/:hk", adminHandler.DeleteCurrency)
	}

	// Log clickable links for server and Swagger
	log.Printf("🚀 Starting server on: \033[1;34mhttp://%s\033[0m\n", serverAddr)
	log.Printf("📄 Swagger documentation available at: \033[1;34m%s\033[0m\n", swaggerURL)

	// Start the server
	srv := &http.Server{
		Addr:    serverAddr,
		Handler: r,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not start server: %v\n", err)
		}
	}()

	// Graceful shutdown setup
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Cleanup base definitions from Redis
	if err := adminHandler.CleanupBaseDefinitions(); err != nil {
		log.Printf("Error during cleanup: %v\n", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server stopped gracefully")
}
