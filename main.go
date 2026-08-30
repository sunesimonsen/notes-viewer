package main

import (
	"log"
	"net/http"

	"github.com/joho/godotenv"
	"github.com/sunesimonsen/notes-viewer/server"
)

func main() {
	// Load .env if present; ignore error when the file does not exist
	// (e.g. production where env vars are set by the platform).
	_ = godotenv.Load()

	config, err := server.ConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	srv, err := server.NewServer(config)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Listening on port %s", config.Port)
	if err := http.ListenAndServe(":"+config.Port, srv); err != nil {
		log.Fatal(err)
	}
}
