package main

import (
	"fmt"
	"io/ioutil"
	collectorrestart "km-agent/CollectorRestart"
	"km-agent/instruement"
	"km-agent/handler"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var manager = &collectorrestart.CollectorManager{}
var server *http.Server

var stopSignal = make(chan bool)

func updateConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	if err := handler.HandleConfig(body); err != nil {
		http.Error(w, "Updating config file failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := manager.Restart(); err != nil {
		log.Fatalf("Failed to restart collector: %v", err)
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Configuration updated and collector restarted successfully")
}

func collectorStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	status := "Collector is not running"
	if manager.Cmd != nil && manager.Cmd.Process != nil && manager.Cmd.ProcessState == nil {
		status = "Collector is running"
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, status)
}

func shutdownHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	func() {
		fmt.Println("Shutdown request received. Stopping services...")
		stopSignal <- true
	}()

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Shutting down collector and HTTP server...")
}

func startServer() {
	server = &http.Server{
		Addr: ":8080",
	}

	http.HandleFunc("/config/update", updateConfigHandler)
	http.HandleFunc("/status", collectorStatus)
	http.HandleFunc("/shutdown", shutdownHandler)

	go func() {
		fmt.Println("Server running on port 8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()
}

func stopServer() {
	if server != nil {
		fmt.Println("Shutting down HTTP server...")
		if err := server.Close(); err != nil {
			log.Fatalf("Failed to shut down server: %v", err)
		}
		fmt.Println("HTTP server stopped")
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Expected 'start' or 'stop' command")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "start":
		fmt.Println("Starting the collector and HTTP server...")
		runStart()
	default:
		fmt.Println("Unknown command. Use 'start' or 'stop'")
		os.Exit(1)
	}
}

func runStart() {
	instrument.StartTelemetry()

	Interrupt := make(chan os.Signal, 1)
	signal.Notify(Interrupt, syscall.SIGINT, syscall.SIGTERM)


	if err := manager.Start(); err != nil {
		log.Fatalf("Failed to start collector: %v", err)
	}

	startServer()

	select {
	case <-Interrupt:
		fmt.Println("Termination signal received. Stopping services...")
	case <-stopSignal:
		fmt.Println("Shutdown endpoint triggered. Stopping services...")
	}

	stopServer()
	if err := manager.Stop(); err != nil {
		log.Fatalf("Failed to stop collector: %v", err)
	}
	instrument.StopTelemetry()

	fmt.Println("All services stopped gracefully")
}


