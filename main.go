package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ak-softwaredev/job-queue-api/worker"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	mux := http.NewServeMux()
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	var workers sync.WaitGroup
	for range 10 {
		workers.Add(1)

		go func() {
			defer workers.Done()
			worker.DoJob()
		}()
	}

	mux.HandleFunc("POST /jobs", worker.PostJob)
	mux.HandleFunc("GET /jobs/{id}", worker.GetJob)
	mux.HandleFunc("DELETE /jobs/{id}", worker.DeleteJob)

	server := &http.Server{
		Addr:    ":" + os.Getenv("PORT"),
		Handler: mux,
	}

	serverErrors := make(chan error, 1)

	fmt.Println("listening on :" + os.Getenv("PORT"))
	go func() {
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		// Shutdown signal.
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
		return
	}

	start := time.Now()
	worker.BeginShutdown()

	shutdown_ctx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	)

	err := server.Shutdown(shutdown_ctx)
	if err != nil {
		log.Printf("HTTP shutdown: %v", err)
		_ = server.Close()
	}
	cancel()

	worker.CloseJobQueue()

	workers.Wait()

	log.Printf("shutdown complete in %v", time.Since(start))

}
