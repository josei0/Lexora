package http

import (
	"context"
	"net/http"
	"time"
)

type Server struct {
	srv *http.Server
}

func NewServer(port string, h http.Handler) *Server {
	return &Server{
		srv: &http.Server{
			Addr:              ":" + port,
			Handler:           h,
			ReadHeaderTimeout: 10 * time.Second,
		},
	}
}

func (s *Server) Start() error {
	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
