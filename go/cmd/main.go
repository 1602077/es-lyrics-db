package main

import (
	"log"
	"net/http"

	"github.com/1602077/es-lyrics-db/pkg/server"
)

func main() {
	router := server.NewRouter()
	log.Println("Starting server on port: 8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
