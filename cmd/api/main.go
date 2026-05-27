package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	log.Printf("servidor iniciado en :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("error al iniciar servidor: %v", err)
	}
}
