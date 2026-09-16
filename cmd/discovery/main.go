package main

import (
	"log"
	"net/http"
	"os"

	"github.com/randy-girard/flynn-discovery/internal/server"
	"github.com/randy-girard/flynn-discovery/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}
	databaseURL := os.Getenv("DATABASE_URL")
	backend, err := store.OpenPostgres(databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer backend.Close()

	srv := server.New(os.Getenv("URL"), backend)
	if _, err := srv.EnsureDefaultCluster(); err != nil {
		log.Fatal(err)
	}
	log.Printf("flynn-discovery listening on :%s url=%s", port, os.Getenv("URL"))
	log.Fatal(http.ListenAndServe(":"+port, srv))
}
