package main

import (
	"context"
	"encrypted-db/config"
	"encrypted-db/internal/db"
	"encrypted-db/internal/handlers/admin"
	"encrypted-db/internal/handlers/public"
	"encrypted-db/internal/handlers/system"
	"encrypted-db/internal/handlers/user"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// Load configuration
	config.LoadConfig()

	// Initialize database connections
	postgresService := db.NewPostgresService()
	redisService := db.NewRedisService()

	// Create handlers with injected dependencies
	systemHandler := system.NewHandler(postgresService, redisService)
	publicHandler := public.NewHandler()
	userHandler := user.NewHandler()
	adminHandler := admin.NewHandler()

	// Define server address using config values
	serverAddr := fmt.Sprintf("%s:%s", config.Config.Server.IP, config.Config.Server.Port)

	// Set up chi router with middlewares
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Empty root handler for "/"
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(""))
	})

	// System endpoints
	r.Get("/healthcheck", systemHandler.HealthCheckHandler)

	// Public endpoints
	r.Route("/public", func(r chi.Router) {
		r.Get("/endpoint1", publicHandler.Endpoint1)
	})

	// User endpoints with multiple methods
	r.Route("/user", func(r chi.Router) {
		r.Get("/profile", userHandler.GetProfile)
		r.Put("/profile", userHandler.UpdateProfile)
	})

	// Admin endpoints
	r.Route("/admin", func(r chi.Router) {
		r.Get("/dashboard", adminHandler.DashboardHandler)
	})

	srv := &http.Server{
		Addr:    serverAddr,
		Handler: r,
	}

	// Start server in a goroutine to allow graceful shutdown
	go func() {
		log.Printf("Starting server on %s\n", serverAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not start server: %v\n", err)
		}
	}()

	// Set up channel to listen for OS interrupt or terminate signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit // Block until signal is received

	log.Println("Shutting down server...")

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped gracefully")
}
