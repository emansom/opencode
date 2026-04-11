package basic

import (
	"context"
	"fmt"
	"net/http"
)

// DefaultPort is the default server port.
const DefaultPort = 8080

const (
	MaxRetries = 3
	MinTimeout = 100
)

// ErrNotFound is returned when a resource is not found.
var ErrNotFound = fmt.Errorf("not found")

var (
	globalLogger Logger
	globalConfig *Config
)

// Config holds server configuration.
type Config struct {
	Port    int    `json:"port" yaml:"port"`
	Host    string `json:"host"`
	Debug   bool   `json:"debug,omitempty"`
	handler http.Handler
}

// Handler processes HTTP requests.
type Handler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
	Close() error
}

// Logger is a basic logging interface.
type Logger interface {
	Info(msg string)
	Error(msg string, err error)
}

// Duration is a named type for time duration.
type Duration int64

// HandlerFunc is a type alias.
type HandlerFunc = func(http.ResponseWriter, *http.Request)

// Server handles HTTP requests.
type Server struct {
	config  *Config
	handler Handler
	logger  Logger
}

// NewServer creates a new server instance.
func NewServer(cfg *Config, h Handler, l Logger) *Server {
	return &Server{config: cfg, handler: h, logger: l}
}

// Start begins listening for requests.
func (s *Server) Start(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.logger.Info(fmt.Sprintf("starting on %s", addr))
	return http.ListenAndServe(addr, s.handler)
}

// Stop gracefully shuts down the server.
func (s *Server) Stop() error {
	s.logger.Info("shutting down")
	return nil
}

// HandleRequest processes a single request.
func HandleRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello")
}

func init() {
	globalLogger = nil
}
