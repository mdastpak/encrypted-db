package main

import (
	"encrypted-db/config"
	"encrypted-db/graph"
	"encrypted-db/graph/generated"

	"encrypted-db/internal/db"
	"log"
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
)

func main() {
	config.LoadConfig()

	// Initialize services
	postgresService := db.NewPostgresService()
	redisService := db.NewRedisService()

	resolver := graph.NewResolver(postgresService, redisService)
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: resolver}))

	http.Handle("/graphql", srv)
	http.Handle("/playground", playground.Handler("GraphQL Playground", "/graphql"))

	addr := config.Config.Server.IP + ":" + config.Config.Server.Port
	log.Printf("connect to http://%s/playground for GraphQL playground", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
