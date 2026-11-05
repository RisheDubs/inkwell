package main

import (
	"log"
	"net/http"
	"time"

	"github.com/RisheDubs/inkwell/apps/api/internal/httpapi"
)

func main() {
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           httpapi.NewRouter(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("api: listening on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
