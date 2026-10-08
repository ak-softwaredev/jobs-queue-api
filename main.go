package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/ak-softwaredev/job-queue-api/worker"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /jobs", worker.PostJob)
	mux.HandleFunc("GET /jobs/{id}", worker.GetJob)
	mux.HandleFunc("DELETE /jobs/{id}", worker.DeleteJob)

	go worker.DoJob()

	fmt.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
