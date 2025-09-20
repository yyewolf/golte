package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"golte/config"
)

// Message represents a received SMS message
// You can extend this struct as needed
// For now, just number and message
// Timestamp can be added if needed

type Message struct {
	Number  string `json:"number"`
	Message string `json:"message"`
}

// MessageStore keeps the last N messages
// Thread-safe for concurrent access

type MessageStore struct {
	mu       sync.Mutex
	messages []Message
	max      int
}

func NewMessageStore(max int) *MessageStore {
	return &MessageStore{max: max}
}

func (s *MessageStore) Add(msg Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.messages) >= s.max {
		s.messages = s.messages[1:]
	}
	s.messages = append(s.messages, msg)
}

func (s *MessageStore) Last() []Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Message(nil), s.messages...)
}

// Server holds API state

type Server struct {
	cfg         *config.Config
	store       *MessageStore
	SendSMSFunc func(number, message string) error
}

func NewServer(cfg *config.Config, store *MessageStore, sendSMS func(number, message string) error) *Server {
	return &Server{cfg: cfg, store: store, SendSMSFunc: sendSMS}
}

func (s *Server) Start(addr string) {
	http.HandleFunc("/messages", s.handleMessages)
	http.HandleFunc("/send", s.handleSend)
	slog.Info("API server starting", slog.String("addr", addr))
	if err := http.ListenAndServe(addr, nil); err != nil {
		slog.Error("API server error", slog.Any("error", err))
	}
}

func (s *Server) authenticate(r *http.Request) bool {
	token := r.Header.Get("Authorization")
	return token == s.cfg.APIToken
}

func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	if !s.authenticate(r) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	msgs := s.store.Last()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msgs)
}

func (s *Server) handleSend(w http.ResponseWriter, r *http.Request) {
	if !s.authenticate(r) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req Message
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid request body"))
		return
	}
	if req.Number == "" || req.Message == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Missing number or message"))
		return
	}
	if err := s.SendSMSFunc(req.Number, req.Message); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to send SMS"))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("SMS sent"))
}
