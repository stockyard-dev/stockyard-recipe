package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/stockyard-dev/stockyard-recipe/internal/server"
	"github.com/stockyard-dev/stockyard-recipe/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9805"
	}
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./recipe-data"
	}

	db, err := store.Open(dataDir)
	if err != nil {
		log.Fatalf("recipe: %v", err)
	}
	defer db.Close()

	srv := server.New(db, server.DefaultLimits())

	fmt.Printf("\n  Recipe — Self-hosted recipe management with ingredient scaling\n  Dashboard:  http://localhost:%s/ui\n  API:        http://localhost:%s/api\n  Questions? hello@stockyard.dev — I read every message\n\n", port, port)
	log.Printf("recipe: listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, srv))
}
