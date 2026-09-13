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
	"encrypted-db/internal/helpers"
	"encrypted-db/routers"
	"encrypted-db/services"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	// Load configuration
	config.LoadConfig()

	// Initialize all services
	services, cleanup, err := services.InitializeServices()
	if err != nil {
		log.Fatalf("Failed to initialize services: %v", err)
	}
	defer cleanup() // Ensure all services are closed on exit

	// Setup router and start the server
	r := setupRouter(services)
	serverAddr := fmt.Sprintf("%s:%s", config.Config.Server.IP, config.Config.Server.Port)
	startServer(r, serverAddr)
}

// startServer starts the HTTP server with graceful shutdown support
func startServer(r *gin.Engine, serverAddr string) {
	srv := &http.Server{
		Addr:    serverAddr,
		Handler: r,
	}

	// Start the server in a goroutine
	go func() {
		log.Printf("🚀 Starting server on: http://%s\n", serverAddr)
		log.Printf("📄 Swagger documentation available at: http://%s/swagger/index.html\n", serverAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not start server: %v\n", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped gracefully")
}

// setupRouter configures all routes and middleware
func setupRouter(services *services.Services) *gin.Engine {
	r := gin.Default()

	// Setup Swagger documentation
	docs.SwaggerInfo.BasePath = "/"
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Root endpoint
	r.GET("/", func(c *gin.Context) {
		helpers.SendResponse(c, http.StatusOK, "Welcome to the Encrypted-DB API", nil)
		return
	})

	// Register routes for different sections
	routers.AdminRoutes(r.Group("/admin"), services)
	routers.PublicRoutes(r.Group("/public"), services)
	routers.SocketRoutes(r.Group("/ws"), services)
	routers.SystemRoutes(r.Group("/system"), services)
	routers.UserRoutes(r.Group("/user"), services)

	// Handle unmatched routes
	r.NoRoute(func(c *gin.Context) {
		helpers.SendResponse(c, http.StatusBadRequest, "Invalid Route", nil)
		return
	})

	return r
}
