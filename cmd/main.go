package main

import (
	"log"
	"net/http"
	"os"

	"github.com/lanthoor/spendly-auth-backend"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/api/v1/integrity/verify", integrity.VerifyIntegrity)

	log.Printf("Listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
