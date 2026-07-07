package httpserver

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type Server struct {
	mux         *http.ServeMux
	address     string
	readTimeout time.Duration
}

func NewServer(address string, readTimeout time.Duration) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)

	return &Server{mux: mux, address: address, readTimeout: readTimeout}
}

// Handle mounts an additional handler onto the server.
func (s *Server) Handle(pattern string, handler http.Handler) {
	s.mux.Handle(pattern, handler)
}

func (s *Server) Run() {
	server := &http.Server{
		Addr:        s.address,
		ReadTimeout: s.readTimeout,
		Handler:     s.mux,
	}

	slog.Info("http server started", "address", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("http server stopped", "error", err)
		os.Exit(1)
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	_, _ = fmt.Fprintf(w, "OK\n")
}
