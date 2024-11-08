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
	"encrypted-db/internal/auth"
	"encrypted-db/internal/db"
	"encrypted-db/internal/handlers/admin"
	"encrypted-db/internal/handlers/public"
	"encrypted-db/internal/handlers/socket"
	"encrypted-db/internal/handlers/system"
	"encrypted-db/internal/handlers/user"
	"encrypted-db/internal/helpers"
	"encrypted-db/internal/models"
	"encrypted-db/internal/rabbitmq"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type InfraHandlers struct {
	Socket *socket.WebSocketHandler
	System *system.SystemHandler
	Admin  *admin.AdminHandler
	Public *public.PublicHandler
	User   *user.UserHandler
}

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

func main() {
	config.LoadConfig()

	// Initialize services
	services, cleanup, err := initializeServices()
	if err != nil {
		log.Fatalf("Failed to initialize services: %v", err)
	}
	defer cleanup() // Ensure cleanup on exit

	// Initialize handlers
	handlers := initializeHandlers(services)

	// Initialize cache
	if err := initializeCache(handlers); err != nil {
		log.Fatalf("Failed to initialize cache: %v", err)
	}

	serverAddr := fmt.Sprintf("%s:%s", config.Config.Server.IP, config.Config.Server.Port)
	swaggerURL := fmt.Sprintf("http://%s/swagger/index.html", serverAddr)

	// Set up gin and Swagger documentation
	r := gin.Default()
	docs.SwaggerInfo.BasePath = "/"
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler)) // Swagger endpoint

	// Root endpoint
	r.GET("/", func(c *gin.Context) {
		helpers.SendResponse(c, http.StatusOK, "Welcome to the Encrypted-DB API", nil)
	})

	// Initialize routes
	RouteHandler(r, handlers)

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
	if err := handlers.Admin.CleanupBaseDefinitions(); err != nil {
		log.Printf("Error during cleanup: %v\n", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server stopped gracefully")
}

func initializeServices() (*models.InfraServices, func(), error) {
	// Initialize PostgreSQL connection
	postgresService := db.NewPostgresService()

	// Initialize Redis connection
	redisService := db.NewRedisService()

	// Initialize RabbitMQ connection
	rabbitMQService := rabbitmq.NewRabbitMQService()

	// Create an InfraServices instance
	services := &models.InfraServices{
		Postgres: postgresService,
		Redis:    redisService,
		RabbitMQ: rabbitMQService,
	}

	// Define a cleanup function to close services
	cleanup := func() {
		postgresService.Close()
		redisService.Close()
		rabbitMQService.Close()
	}

	return services, cleanup, nil
}

func initializeHandlers(services *models.InfraServices) *InfraHandlers {
	return &InfraHandlers{
		Socket: socket.NewWebSocketHandler(services),
		System: system.NewHandler(services),
		Admin:  admin.NewHandler(services),
		Public: public.NewHandler(services),
		User:   user.NewHandler(services),
	}
}

func initializeCache(ih *InfraHandlers) error {
	err := ih.Admin.LoadAndCacheCurrencies()
	if err != nil {
		return fmt.Errorf("failed to load and cache currencies: %v", err)
	}
	return nil
}

func RouteHandler(r *gin.Engine, ih *InfraHandlers) {
	// WebSocket route for currencies
	socketGroup := r.Group("/ws")
	{
		socketGroup.GET("/", ih.Socket.ServeWSGin)
	}

	// System routes
	systemGroup := r.Group("/system")
	{
		systemGroup.GET("/healthcheck", ih.System.HealthCheckHandler)
		systemGroup.GET("/ping", ih.System.PingPongHandler)
	}

	// Public routes
	publicGroup := r.Group("/public")
	{
		publicGroup.GET("/currencies", ih.Public.GetActiveCurrencies)
		publicGroup.GET("/currencies/:hk", ih.Public.GetCurrencyByHK)
	}

	// Group for admin routes with JWTAdminVerification middleware
	adminGroup := r.Group("/admin")
	adminGroup.Use(auth.JWTVerification)
	{
		adminGroup.POST("/currencies", ih.Admin.CreateCurrency)
		adminGroup.PUT("/currencies/:hk", ih.Admin.UpdateCurrency)
		adminGroup.DELETE("/currencies/:hk", ih.Admin.DeleteCurrency)
	}

	// Group for user routes with JWTUserVerification middleware
	userGroup := r.Group("/user")
	userGroup.Use(auth.JWTVerification)
	{
		userGroup.GET("/profile", nil)
		userGroup.POST("/update", nil)
	}
}
