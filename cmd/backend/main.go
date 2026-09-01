package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const itemCount = 10_000

type message struct {
	Author string `json:"author"`
	Text   string `json:"text"`
}

type historyResponse struct {
	ServerTime string    `json:"server_time"`
	Messages   []message `json:"messages"`
}

type generateResponse struct {
	Bots []string `json:"bots"`
}

type server struct {
	instanceID string
	history    []message
}

func main() {
	instanceID := os.Getenv("INSTANCE_ID")
	if instanceID == "" {
		instanceID = "local"
	}

	s := &server{
		instanceID: instanceID,
		history:    makeHistory(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/history", s.historyHandler)
	mux.HandleFunc("POST /api/generate", s.generateHandler)
	mux.HandleFunc("GET /health", s.healthHandler)
	
	handler := http.Handler(mux)

	if os.Getenv("REQUEST_LOGS") != "false" {
		handler = s.logRequests(handler)
	}

	httpServer := &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("instance=%s listening on %s", instanceID, httpServer.Addr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func makeHistory() []message {
	messages := make([]message, itemCount)
	for i := range messages {
		messages[i] = message{
			Author: "admin",
			Text:   "Welcome to HighloadGram!",
		}
	}
	return messages
}

func (s *server) historyHandler(w http.ResponseWriter, _ *http.Request) {
	response := historyResponse{
		ServerTime: time.Now().UTC().Format(time.RFC3339Nano),
		Messages:   s.history,
	}

	s.writeJSON(w, http.StatusOK, response)
}

func (s *server) generateHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}

	prefix := strings.TrimSpace(r.FormValue("bot_prefix"))
	if prefix == "" {
		prefix = "bot"
	}

	bots := make([]string, itemCount)
	for i := range bots {
		bots[i] = fmt.Sprintf("%s_%d", prefix, i+1)
	}

	s.writeJSON(w, http.StatusOK, generateResponse{Bots: bots})
}

func (s *server) healthHandler(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{
		"status":   "ok",
		"instance": s.instanceID,
	})
}

func (s *server) writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Backend-Instance", s.instanceID)
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("instance=%s encode response: %v", s.instanceID, err)
	}
}

func (s *server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		next.ServeHTTP(w, r)
		log.Printf(
			"instance=%s method=%s path=%s duration=%s",
			s.instanceID,
			r.Method,
			r.URL.Path,
			time.Since(startedAt),
		)
	})
}
