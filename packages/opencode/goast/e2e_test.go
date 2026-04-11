package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ============================================================================
// Part 1: AST Tools — Complex Multi-File Go Project
// ============================================================================

// scaffold creates a realistic multi-file Go project in a temp directory.
func scaffold(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	files := map[string]string{
		"go.mod": "module github.com/example/webapp\n\ngo 1.22\n",
		"main.go": `package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	return nil
}
`,
		"internal/config/config.go": `package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds the application configuration.
type Config struct {
	Port         int           ` + "`" + `yaml:"port" json:"port"` + "`" + `
	Host         string        ` + "`" + `yaml:"host" json:"host"` + "`" + `
	ReadTimeout  time.Duration ` + "`" + `yaml:"read_timeout"` + "`" + `
	WriteTimeout time.Duration ` + "`" + `yaml:"write_timeout"` + "`" + `
	DatabaseURL  string        ` + "`" + `yaml:"database_url"` + "`" + `
	JWTSecret    string        ` + "`" + `yaml:"jwt_secret" json:"-"` + "`" + `
}

// DatabaseConfig holds database-specific settings.
type DatabaseConfig struct {
	Host         string ` + "`" + `yaml:"host"` + "`" + `
	Port         int    ` + "`" + `yaml:"port"` + "`" + `
	Name         string ` + "`" + `yaml:"name"` + "`" + `
	User         string ` + "`" + `yaml:"user"` + "`" + `
	Password     string ` + "`" + `yaml:"password"` + "`" + `
	MaxOpenConns int    ` + "`" + `yaml:"max_open_conns"` + "`" + `
}

const (
	DefaultPort = 8080
	DefaultHost = "localhost"
)

func Load(path string) (*Config, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("config file not found: %w", err)
	}
	return &Config{Port: DefaultPort, Host: DefaultHost}, nil
}

func (c *Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("database URL is required")
	}
	return nil
}
`,
		"internal/storage/store.go": `package storage

import (
	"context"
	"time"
)

type UserRecord struct {
	ID        int64
	Name      string
	Email     string
	Role      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Store interface {
	ListUsers(ctx context.Context, page int, pageSize int) ([]*UserRecord, int, error)
	GetUser(ctx context.Context, id int64) (*UserRecord, error)
	CreateUser(ctx context.Context, user *UserRecord) (*UserRecord, error)
	UpdateUser(ctx context.Context, user *UserRecord) (*UserRecord, error)
	DeleteUser(ctx context.Context, id int64) error
	Ping(ctx context.Context) error
	Close() error
}
`,
		"internal/storage/postgres.go": `package storage

import (
	"context"
	"fmt"
)

type PostgresStore struct {
	connString string
}

func NewPostgresStore(connString string) (*PostgresStore, error) {
	if connString == "" {
		return nil, fmt.Errorf("connection string is required")
	}
	return &PostgresStore{connString: connString}, nil
}

func (p *PostgresStore) ListUsers(ctx context.Context, page int, pageSize int) ([]*UserRecord, int, error) {
	return nil, 0, fmt.Errorf("not implemented")
}

func (p *PostgresStore) GetUser(ctx context.Context, id int64) (*UserRecord, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p *PostgresStore) CreateUser(ctx context.Context, user *UserRecord) (*UserRecord, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p *PostgresStore) UpdateUser(ctx context.Context, user *UserRecord) (*UserRecord, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p *PostgresStore) DeleteUser(ctx context.Context, id int64) error {
	return fmt.Errorf("not implemented")
}

func (p *PostgresStore) Ping(ctx context.Context) error {
	return nil
}

func (p *PostgresStore) Close() error {
	return nil
}
`,
		"internal/server/server.go": `package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
)

type Server struct {
	cfg      interface{}
	router   http.Handler
	listener net.Listener
}

func New(cfg interface{}) *Server {
	return &Server{cfg: cfg}
}

func (s *Server) Start(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", 8080)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	s.listener = ln
	return http.Serve(ln, s.router)
}

func (s *Server) Addr() string {
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}
`,
		"internal/server/handlers.go": `package server

import (
	"encoding/json"
	"net/http"
	"time"
)

type HealthResponse struct {
	Status string ` + "`" + `json:"status"` + "`" + `
	Time   string ` + "`" + `json:"time"` + "`" + `
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HealthResponse{
		Status: "ok",
		Time:   time.Now().UTC().Format(time.RFC3339),
	})
}
`,
		"internal/auth/auth.go": `package auth

import (
	"errors"
	"time"
)

var (
	ErrInvalidToken  = errors.New("invalid token")
	ErrExpiredToken  = errors.New("token has expired")
)

type Claims struct {
	UserID    int64
	Email     string
	Role      string
	ExpiresAt time.Time
	IssuedAt  time.Time
}

type TokenService struct {
	secret     []byte
	expiration time.Duration
}

func NewTokenService(secret string, expiration time.Duration) *TokenService {
	return &TokenService{
		secret:     []byte(secret),
		expiration: expiration,
	}
}

func (t *TokenService) GenerateToken(claims *Claims) (string, error) {
	if claims.UserID == 0 {
		return "", errors.New("user ID is required")
	}
	return "token-placeholder", nil
}

func (t *TokenService) ValidateToken(token string) (*Claims, error) {
	if token == "" {
		return nil, ErrInvalidToken
	}
	return nil, ErrExpiredToken
}
`,
		"internal/middleware/middleware.go": `package middleware

import (
	"log"
	"net/http"
	"time"
)

func Chain(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic: %v", err)
				http.Error(w, "internal server error", 500)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
`,
	}

	for name, content := range files {
		p := filepath.Join(dir, name)
		os.MkdirAll(filepath.Dir(p), 0755)
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

// editAndPersist runs an edit operation and writes the result back to disk.
func editAndPersist(t *testing.T, op Operation) *EditResult {
	t.Helper()
	r, err := edit(op)
	if err != nil {
		t.Fatalf("edit %s on %s: %v", op.Op, op.File, err)
	}
	if r.Content == "" {
		t.Fatalf("edit %s on %s: no content returned", op.Op, op.File)
	}
	if err := os.WriteFile(op.File, []byte(r.Content), 0644); err != nil {
		t.Fatalf("write back %s: %v", op.File, err)
	}
	return r
}

// inspectFile runs inspect on a file and returns the result.
func inspectFile(t *testing.T, path string) *InspectResult {
	t.Helper()
	r, err := inspect(path)
	if err != nil {
		t.Fatalf("inspect %s: %v", path, err)
	}
	return r
}

// ---- Test: Inspect all files in the project ----

func TestE2EInspectAllFiles(t *testing.T) {
	dir := scaffold(t)

	files := []struct {
		path    string
		pkg     string
		minFunc int
		minType int
	}{
		{"main.go", "main", 2, 0},
		{"internal/config/config.go", "config", 2, 2},
		{"internal/storage/store.go", "storage", 0, 1},
		{"internal/storage/postgres.go", "storage", 8, 1},
		{"internal/server/server.go", "server", 3, 1},
		{"internal/server/handlers.go", "server", 1, 1},
		{"internal/auth/auth.go", "auth", 3, 2},
		{"internal/middleware/middleware.go", "middleware", 3, 0},
	}

	for _, f := range files {
		t.Run(f.path, func(t *testing.T) {
			r := inspectFile(t, filepath.Join(dir, f.path))
			if r.Package != f.pkg {
				t.Errorf("package: got %q, want %q", r.Package, f.pkg)
			}
			if len(r.Functions) < f.minFunc {
				t.Errorf("functions: got %d, want >= %d", len(r.Functions), f.minFunc)
			}
			if len(r.Types) < f.minType {
				t.Errorf("types: got %d, want >= %d", len(r.Types), f.minType)
			}
		})
	}
}

// ---- Test: Cross-file struct field additions ----

func TestE2ECrossFileStructEdits(t *testing.T) {
	dir := scaffold(t)
	configPath := filepath.Join(dir, "internal/config/config.go")

	// Add a LogLevel field to Config
	editAndPersist(t, Operation{File: configPath, Mode: "edit", Op: "add_struct_field",
		Target: "Config", FieldName: "LogLevel", FieldType: "string",
		Tag: `yaml:"log_level" json:"log_level"`})

	// Add a MaxRetries field after LogLevel
	editAndPersist(t, Operation{File: configPath, Mode: "edit", Op: "add_struct_field",
		Target: "Config", FieldName: "MaxRetries", FieldType: "int",
		Tag: `yaml:"max_retries"`, Position: "after", Anchor: "LogLevel"})

	// Verify
	r := inspectFile(t, configPath)
	var configType *TypeInfo
	for i := range r.Types {
		if r.Types[i].Name == "Config" {
			configType = &r.Types[i]
		}
	}
	if configType == nil {
		t.Fatal("Config type not found")
	}
	fieldNames := make([]string, len(configType.Fields))
	for i, f := range configType.Fields {
		fieldNames[i] = f.Name
	}
	// LogLevel should be before MaxRetries
	logIdx, maxIdx := -1, -1
	for i, n := range fieldNames {
		if n == "LogLevel" {
			logIdx = i
		}
		if n == "MaxRetries" {
			maxIdx = i
		}
	}
	if logIdx < 0 {
		t.Error("LogLevel field not found")
	}
	if maxIdx < 0 {
		t.Error("MaxRetries field not found")
	}
	if logIdx >= 0 && maxIdx >= 0 && maxIdx != logIdx+1 {
		t.Errorf("MaxRetries should be right after LogLevel, got fields: %v", fieldNames)
	}
}

// ---- Test: Add interface + implement it ----

func TestE2EAddInterfaceAndImplement(t *testing.T) {
	dir := scaffold(t)
	storePath := filepath.Join(dir, "internal/storage/store.go")
	pgPath := filepath.Join(dir, "internal/storage/postgres.go")

	// Create a CacheStore interface
	editAndPersist(t, Operation{File: storePath, Mode: "edit", Op: "create_interface",
		Name: "CacheStore"})

	// Add methods to it
	editAndPersist(t, Operation{File: storePath, Mode: "edit", Op: "add_interface_method",
		Target: "CacheStore", MethodName: "Get", Params: "ctx:context.Context,key:string",
		Returns: "interface{},bool"})

	editAndPersist(t, Operation{File: storePath, Mode: "edit", Op: "add_interface_method",
		Target: "CacheStore", MethodName: "Set",
		Params: "ctx:context.Context,key:string,value:interface{}", Returns: "error"})

	editAndPersist(t, Operation{File: storePath, Mode: "edit", Op: "add_interface_method",
		Target: "CacheStore", MethodName: "Delete",
		Params: "ctx:context.Context,key:string", Returns: "error"})

	// Verify interface has 3 methods
	r := inspectFile(t, storePath)
	var cacheIface *TypeInfo
	for i := range r.Types {
		if r.Types[i].Name == "CacheStore" {
			cacheIface = &r.Types[i]
		}
	}
	if cacheIface == nil {
		t.Fatal("CacheStore not found")
	}
	if len(cacheIface.Methods) != 3 {
		t.Errorf("want 3 methods, got %d", len(cacheIface.Methods))
	}

	// Now implement Get on PostgresStore
	editAndPersist(t, Operation{File: pgPath, Mode: "edit", Op: "create_method",
		ReceiverType: "*PostgresStore", ReceiverVar: "p", Name: "Get",
		Params: "ctx:context.Context,key:string", Returns: "interface{},bool"})

	// Add body: return nil, false
	editAndPersist(t, Operation{File: pgPath, Mode: "edit", Op: "insert_return",
		Target: "PostgresStore.Get", Values: "nil,false"})

	// Verify
	rPg := inspectFile(t, pgPath)
	found := false
	for _, f := range rPg.Functions {
		if f.Name == "PostgresStore.Get" {
			found = true
			if len(f.Body) < 1 {
				t.Error("Get method body is empty")
			}
		}
	}
	if !found {
		t.Error("PostgresStore.Get not found")
	}
}

// ---- Test: Build a complete function step by step ----

func TestE2EBuildCompleteFunction(t *testing.T) {
	dir := scaffold(t)
	serverPath := filepath.Join(dir, "internal/server/server.go")

	// Create Shutdown method
	editAndPersist(t, Operation{File: serverPath, Mode: "edit", Op: "create_method",
		ReceiverType: "*Server", ReceiverVar: "s", Name: "Shutdown",
		Params: "ctx:context.Context", Returns: "error"})

	// if s.listener == nil { return nil }
	editAndPersist(t, Operation{File: serverPath, Mode: "edit", Op: "insert_if",
		Target: "Server.Shutdown", CondLeft: "selector:s.listener", Operator: "==",
		CondRight: "nil", Position: "first"})

	editAndPersist(t, Operation{File: serverPath, Mode: "edit", Op: "insert_return",
		Target: "Server.Shutdown.if[0]", Values: "nil", Position: "first"})

	// err := s.listener.Close()
	editAndPersist(t, Operation{File: serverPath, Mode: "edit", Op: "insert_assign_call",
		Target: "Server.Shutdown", Vars: "err", Short: boolPtr(true),
		Receiver: "s.listener", Method: "Close", Position: "last"})

	// return err
	editAndPersist(t, Operation{File: serverPath, Mode: "edit", Op: "insert_return",
		Target: "Server.Shutdown", Values: "ident:err", Position: "last"})

	// Verify final content
	content, _ := os.ReadFile(serverPath)
	s := string(content)
	if !strings.Contains(s, "func (s *Server) Shutdown(ctx context.Context) error") {
		t.Error("Shutdown signature not found")
	}
	if !strings.Contains(s, "if s.listener == nil") {
		t.Error("nil check not found")
	}
	if !strings.Contains(s, "return nil") {
		t.Error("return nil not found in if body")
	}
	if !strings.Contains(s, "err := s.listener.Close()") {
		t.Error("Close call not found")
	}
	if !strings.Contains(s, "return err") {
		t.Error("return err not found")
	}
}

// ---- Test: Add error handling to existing function ----

func TestE2EAddErrorHandling(t *testing.T) {
	dir := scaffold(t)
	authPath := filepath.Join(dir, "internal/auth/auth.go")

	// Add a RefreshToken method
	editAndPersist(t, Operation{File: authPath, Mode: "edit", Op: "create_method",
		ReceiverType: "*TokenService", ReceiverVar: "t", Name: "RefreshToken",
		Params: "oldToken:string", Returns: "string,error"})

	// claims, err := t.ValidateToken(oldToken)
	editAndPersist(t, Operation{File: authPath, Mode: "edit", Op: "insert_assign_call",
		Target: "TokenService.RefreshToken", Vars: "claims,err", Short: boolPtr(true),
		Receiver: "t", Method: "ValidateToken", Args: "ident:oldToken", Position: "first"})

	// if err != nil { return "", err }
	editAndPersist(t, Operation{File: authPath, Mode: "edit", Op: "insert_error_check",
		Target: "TokenService.RefreshToken", ErrVar: "err",
		ReturnValues: "string:,ident:err", Position: "at_index", Index: intPtr(1)})

	// return t.GenerateToken(claims)
	editAndPersist(t, Operation{File: authPath, Mode: "edit", Op: "insert_assign_call",
		Target: "TokenService.RefreshToken", Vars: "newToken,err", Short: boolPtr(true),
		Receiver: "t", Method: "GenerateToken", Args: "ident:claims", Position: "last"})

	editAndPersist(t, Operation{File: authPath, Mode: "edit", Op: "insert_error_check",
		Target: "TokenService.RefreshToken", ErrVar: "err",
		ReturnValues: "string:,ident:err", Position: "last"})

	editAndPersist(t, Operation{File: authPath, Mode: "edit", Op: "insert_return",
		Target: "TokenService.RefreshToken", Values: "ident:newToken,nil", Position: "last"})

	// Verify
	content, _ := os.ReadFile(authPath)
	s := string(content)
	if !strings.Contains(s, "claims, err := t.ValidateToken(oldToken)") {
		t.Errorf("ValidateToken call not found, got:\n%s", s)
	}
	if strings.Count(s, "if err != nil") < 2 {
		t.Error("expected at least 2 error checks in RefreshToken")
	}
}

// ---- Test: Modify struct tags ----

func TestE2EModifyStructTags(t *testing.T) {
	dir := scaffold(t)
	configPath := filepath.Join(dir, "internal/config/config.go")

	// Add a "validate" tag to Port field
	editAndPersist(t, Operation{File: configPath, Mode: "edit", Op: "set_struct_tag",
		Target: "Config.Port", TagKey: "validate", TagValue: "required,min=1,max=65535"})

	// Add a "validate" tag to Host field
	editAndPersist(t, Operation{File: configPath, Mode: "edit", Op: "set_struct_tag",
		Target: "Config.Host", TagKey: "validate", TagValue: "required"})

	// Verify original tags are preserved
	r := inspectFile(t, configPath)
	var cfg *TypeInfo
	for i := range r.Types {
		if r.Types[i].Name == "Config" {
			cfg = &r.Types[i]
		}
	}
	if cfg == nil {
		t.Fatal("Config not found")
	}
	for _, f := range cfg.Fields {
		if f.Name == "Port" {
			if !strings.Contains(f.Tag, "yaml:") {
				t.Error("Port: yaml tag lost")
			}
			if !strings.Contains(f.Tag, "json:") {
				t.Error("Port: json tag lost")
			}
			if !strings.Contains(f.Tag, "validate:") {
				t.Error("Port: validate tag not added")
			}
		}
		if f.Name == "Host" {
			if !strings.Contains(f.Tag, "yaml:") {
				t.Error("Host: yaml tag lost")
			}
			if !strings.Contains(f.Tag, "validate:") {
				t.Error("Host: validate tag not added")
			}
		}
	}
}

// ---- Test: Add imports and const block ----

func TestE2EImportsAndConsts(t *testing.T) {
	dir := scaffold(t)
	serverPath := filepath.Join(dir, "internal/server/server.go")

	// Add a time import
	editAndPersist(t, Operation{File: serverPath, Mode: "edit", Op: "add_import",
		Path: "time"})

	// Add a const
	editAndPersist(t, Operation{File: serverPath, Mode: "edit", Op: "add_const",
		Name: "DefaultShutdownTimeout", Type: "time.Duration",
		ValueSpec: "call:time.Second"})

	// Verify
	r := inspectFile(t, serverPath)
	foundImport := false
	for _, imp := range r.Imports {
		if imp.Path == "time" {
			foundImport = true
		}
	}
	if !foundImport {
		t.Error("time import not found")
	}
	foundConst := false
	for _, c := range r.Consts {
		if c.Name == "DefaultShutdownTimeout" {
			foundConst = true
		}
	}
	if !foundConst {
		t.Error("DefaultShutdownTimeout const not found")
	}
}

// ---- Test: Rename and refactor ----

func TestE2ERename(t *testing.T) {
	dir := scaffold(t)
	authPath := filepath.Join(dir, "internal/auth/auth.go")

	// Rename ErrInvalidToken to ErrBadToken
	editAndPersist(t, Operation{File: authPath, Mode: "edit", Op: "rename",
		Target: "ErrInvalidToken", NewName: "ErrBadToken"})

	// Verify all references updated
	content, _ := os.ReadFile(authPath)
	s := string(content)
	if strings.Contains(s, "ErrInvalidToken") {
		t.Error("old name ErrInvalidToken still present")
	}
	if !strings.Contains(s, "ErrBadToken") {
		t.Error("new name ErrBadToken not found")
	}
	// The var declaration and usage in ValidateToken should both be updated
	if strings.Count(s, "ErrBadToken") < 2 {
		t.Errorf("expected ErrBadToken in at least 2 places (decl + usage), found %d", strings.Count(s, "ErrBadToken"))
	}
}

// ---- Test: Delete declaration ----

func TestE2EDeleteDeclaration(t *testing.T) {
	dir := scaffold(t)
	configPath := filepath.Join(dir, "internal/config/config.go")

	// Delete DatabaseConfig type
	editAndPersist(t, Operation{File: configPath, Mode: "edit", Op: "delete",
		Target: "DatabaseConfig"})

	// Verify it's gone
	r := inspectFile(t, configPath)
	for _, ty := range r.Types {
		if ty.Name == "DatabaseConfig" {
			t.Error("DatabaseConfig should have been deleted")
		}
	}
	// Config should still exist
	found := false
	for _, ty := range r.Types {
		if ty.Name == "Config" {
			found = true
		}
	}
	if !found {
		t.Error("Config type was accidentally deleted")
	}
}

// ---- Test: Extract interface from struct methods ----

func TestE2EExtractInterface(t *testing.T) {
	dir := scaffold(t)
	authPath := filepath.Join(dir, "internal/auth/auth.go")

	// Extract interface from TokenService's exported methods
	editAndPersist(t, Operation{File: authPath, Mode: "edit", Op: "extract_interface",
		Target: "TokenService", InterfaceName: "Authenticator"})

	// Verify interface exists with the right methods
	r := inspectFile(t, authPath)
	var iface *TypeInfo
	for i := range r.Types {
		if r.Types[i].Name == "Authenticator" {
			iface = &r.Types[i]
		}
	}
	if iface == nil {
		t.Fatal("Authenticator interface not found")
	}
	if iface.Kind != "interface" {
		t.Errorf("expected interface, got %s", iface.Kind)
	}
	methodNames := make([]string, len(iface.Methods))
	for i, m := range iface.Methods {
		methodNames[i] = m.Name
	}
	if len(methodNames) < 2 {
		t.Errorf("expected at least 2 methods (GenerateToken, ValidateToken), got %v", methodNames)
	}
}

// ---- Test: Doc comments ----

func TestE2EDocComments(t *testing.T) {
	dir := scaffold(t)
	storePath := filepath.Join(dir, "internal/storage/store.go")

	// Add doc comment to Store interface
	editAndPersist(t, Operation{File: storePath, Mode: "edit", Op: "set_doc_comment",
		Target: "Store", Text: "Store defines the data access interface for the application."})

	// Add doc comment to UserRecord
	editAndPersist(t, Operation{File: storePath, Mode: "edit", Op: "set_doc_comment",
		Target: "UserRecord", Text: "UserRecord represents a user entity in the database."})

	// Verify
	r := inspectFile(t, storePath)
	for _, ty := range r.Types {
		if ty.Name == "Store" && !strings.Contains(ty.Doc, "data access interface") {
			t.Error("Store doc comment not set")
		}
		if ty.Name == "UserRecord" && !strings.Contains(ty.Doc, "user entity") {
			t.Error("UserRecord doc comment not set")
		}
	}
}

// ---- Test: go_fix on broken file ----

func TestE2EFixBrokenFile(t *testing.T) {
	dir := t.TempDir()

	// Test 1: fully fixable errors (double semicolons, statements on same line as brace)
	path1 := filepath.Join(dir, "fixable.go")
	os.WriteFile(path1, []byte(`package main

import "fmt"

func main() {
	x := 1;;
	fmt.Println(x)
}
`), 0644)

	result, err := fix(path1)
	if err != nil {
		t.Fatalf("fix fixable: %v", err)
	}
	if len(result.FixesApplied) == 0 {
		t.Error("expected fixes to be applied")
	}
	content, _ := os.ReadFile(path1)
	errs := parseErrors(path1, content)
	if len(errs) > 0 {
		t.Errorf("fixable file still has errors: %v", errs)
	}

	// Test 2: partially fixable (double semicolons fixed, but orphaned statement can't be auto-fixed)
	path2 := filepath.Join(dir, "partial.go")
	os.WriteFile(path2, []byte(`package main

import "fmt"

func main() {
	x := 1;;
	fmt.Println(x)
}    y := 2
`), 0644)

	result2, err := fix(path2)
	if err != nil {
		t.Fatalf("fix partial: %v", err)
	}
	if len(result2.FixesApplied) == 0 {
		t.Error("expected some fixes")
	}
	// Should have fewer errors than before (;; fixed, brace split done)
	// but y := 2 at package level is genuinely unfixable
	if len(result2.RemainingErrs) == 0 {
		t.Error("expected remaining errors for orphaned statement")
	}
	t.Logf("fixes: %v, remaining: %v", result2.FixesApplied, result2.RemainingErrs)
}

// ============================================================================
// Part 2: llama.cpp Server Integration (Gemma 4 E2B model)
// ============================================================================

// These tests require a llama.cpp server running at localhost:8080
// with a Gemma 4 model loaded. Skip if not available.

const llamaServerURL = "http://localhost:8080"
const gemmaE2BModel = "ggml-org/gemma-4-E2B-it-GGUF:F16"

type chatMessage struct {
	Role             string     `json:"role"`
	Content          string     `json:"content,omitempty"`
	ToolCalls        []toolCall `json:"tool_calls,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
}

type toolCall struct {
	Type     string       `json:"type"`
	Function toolCallFunc `json:"function"`
	ID       string       `json:"id"`
}

type toolCallFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type toolDef struct {
	Type     string      `json:"type"`
	Function toolDefFunc `json:"function"`
}

type toolDefFunc struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Tools       []toolDef     `json:"tools,omitempty"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type chatResponse struct {
	Choices []struct {
		FinishReason string      `json:"finish_reason"`
		Message      chatMessage `json:"message"`
	} `json:"choices"`
	Model            string `json:"model"`
	SystemFingerprint string `json:"system_fingerprint"`
}

func llamaAvailable() bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(llamaServerURL + "/v1/models")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 200
}

func sendChat(t *testing.T, req chatRequest) chatResponse {
	t.Helper()
	body, _ := json.Marshal(req)
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Post(llamaServerURL+"/v1/chat/completions", "application/json",
		strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("send chat: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		t.Fatalf("chat error %d: %s", resp.StatusCode, string(respBody))
	}
	var cr chatResponse
	if err := json.Unmarshal(respBody, &cr); err != nil {
		t.Fatalf("parse response: %v\nbody: %s", err, string(respBody))
	}
	return cr
}

// goAstSystemPrompt returns the system prompt that teaches Gemma 4 how to use the tools.
// This mirrors what OpenCode sends via buildGemma4SystemPrompt() + generateGoAstSystemPrompt().
// goAstSystemPrompt returns the system prompt generated by the Go helper's
// operation registry — the single source of truth for tool descriptions.
// This is the same prompt that OpenCode uses at runtime.
func goAstSystemPrompt() string {
	result := generatePrompt()
	return `You are an expert Go programming assistant. You modify Go source files using AST-based tools.
Always call go_inspect first to understand the file structure before editing.
Each tool performs one AST operation — call one tool at a time.

` + result.SystemPrompt
}

// goAstTools returns tool definitions from the operation registry.
// The registry is the single source of truth for tool names, descriptions,
// and parameters — shared by tests, OpenCode TypeScript, and the system prompt.
// This matches Gemma 4's FC (Function Calling) format training: one tool
// per action, flat human-readable parameters, no dispatcher "op" field.
func goAstTools() []toolDef {
	prompt := generatePrompt()
	tools := make([]toolDef, 0, len(prompt.Tools)+2)

	// Convert each registry entry to the OpenAI tool definition format
	for _, info := range prompt.Tools {
		props := map[string]interface{}{
			"filePath": map[string]interface{}{"type": "string", "description": "Absolute path to the .go file"},
		}
		required := []string{"filePath"}
		for _, p := range info.Params {
			prop := map[string]interface{}{"type": p.Type, "description": p.Description}
			if len(p.Enum) > 0 {
				prop["enum"] = p.Enum
			}
			props[p.Name] = prop
			if p.Required {
				required = append(required, p.Name)
			}
		}
		tools = append(tools, toolDef{Type: "function", Function: toolDefFunc{
			Name:        info.Name,
			Description: info.Description,
			Parameters: map[string]interface{}{
				"type": "object", "properties": props, "required": required,
			},
		}})
	}

	// go_inspect and go_fix are not edit operations — define them directly
	tools = append(tools, toolDef{Type: "function", Function: toolDefFunc{
		Name:        "go_inspect",
		Description: "Inspect the AST structure of a Go source file. Returns package name, imports, functions, types, and variables. Always call this before editing.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"filePath": map[string]interface{}{"type": "string", "description": "Absolute path to the .go file"},
			},
			"required": []string{"filePath"},
		},
	}})
	tools = append(tools, toolDef{Type: "function", Function: toolDefFunc{
		Name:        "go_fix",
		Description: "Automatically fix syntax errors in a Go source file. Call when an edit fails with parse errors.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"filePath": map[string]interface{}{"type": "string", "description": "Absolute path to the .go file"},
			},
			"required": []string{"filePath"},
		},
	}})
	return tools
}

// blockedTools returns tools that should NOT be used for .go files
func blockedTools() []toolDef {
	return []toolDef{
		{Type: "function", Function: toolDefFunc{
			Name:        "edit",
			Description: "Text-based search and replace in a file. Cannot be used on .go files.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"filePath":  map[string]interface{}{"type": "string"},
					"oldString": map[string]interface{}{"type": "string"},
					"newString": map[string]interface{}{"type": "string"},
				},
				"required": []string{"filePath", "oldString", "newString"},
			},
		}},
		{Type: "function", Function: toolDefFunc{
			Name:        "write",
			Description: "Write content to a file. Cannot be used to modify existing .go files.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"filePath": map[string]interface{}{"type": "string"},
					"content":  map[string]interface{}{"type": "string"},
				},
				"required": []string{"filePath", "content"},
			},
		}},
	}
}

// parseToolCallOp converts a model's tool call into an Operation.
// The tool name gives the op directly (e.g. "go_create_function" → "create_function").
// Falls back to ParseToolCallJSON for normalization of confused parameter patterns.
func parseToolCallOp(t *testing.T, tc toolCall, overrideFile string) Operation {
	t.Helper()
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &raw); err != nil {
		t.Fatalf("parse tool call args: %v\nraw: %s", err, tc.Function.Arguments)
	}

	op := ParseToolCallJSON(raw)

	// Tool name → op: strip "go_" prefix. This is the primary mechanism
	// in the FC format — each tool name uniquely identifies the operation.
	if strings.HasPrefix(tc.Function.Name, "go_") {
		toolOp := strings.TrimPrefix(tc.Function.Name, "go_")
		if KnownOps[toolOp] {
			op.Op = toolOp
		}
	}

	if overrideFile != "" {
		op.File = overrideFile
	}
	NormalizeOp(&op)

	t.Logf("  parsed op: %s target=%q name=%q receiverType=%q receiver=%q method=%q",
		op.Op, op.Target, op.Name, op.ReceiverType, op.Receiver, op.Method)
	return op
}

// executeModelToolCall runs the model's tool call through the Go AST helper
// and writes the result back to disk. Returns the edit result.
func executeModelToolCall(t *testing.T, tc toolCall, filePath string) *EditResult {
	t.Helper()
	op := parseToolCallOp(t, tc, filePath)
	r, err := edit(op)
	if err != nil {
		t.Fatalf("execute model's %s call: %v", op.Op, err)
	}
	if r.Content != "" {
		if err := os.WriteFile(filePath, []byte(r.Content), 0644); err != nil {
			t.Fatalf("write result: %v", err)
		}
	}
	return r
}

// requireToolCall extracts the first tool call from a response, failing if none.
func requireToolCall(t *testing.T, resp chatResponse) toolCall {
	t.Helper()
	if len(resp.Choices) == 0 {
		t.Fatal("no choices in response")
	}
	msg := resp.Choices[0].Message
	if len(msg.ToolCalls) == 0 {
		t.Fatalf("expected tool call, got text: %q (reasoning: %s)",
			msg.Content, truncate(msg.ReasoningContent, 300))
	}
	return msg.ToolCalls[0]
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// isGoEditTool returns true if the tool call is a go_* edit operation
// (not go_inspect or go_fix).
func isGoEditTool(name string) bool {
	if !strings.HasPrefix(name, "go_") {
		return false
	}
	return name != "go_inspect" && name != "go_fix"
}

// TestLlamaInspectE2E: model calls go_inspect on a real file, we execute it.
func TestLlamaInspectE2E(t *testing.T) {
	if !llamaAvailable() {
		t.Skip("llama.cpp server not available at " + llamaServerURL)
	}
	dir := scaffold(t)
	serverFile := filepath.Join(dir, "internal/server/server.go")

	resp := sendChat(t, chatRequest{
		Model: gemmaE2BModel,
		Messages: []chatMessage{
			{Role: "system", Content: goAstSystemPrompt()},
			{Role: "user", Content: "Inspect the Go file at " + serverFile},
		},
		Tools:       goAstTools(),
		Temperature: 0.1,
		MaxTokens:   256,
	})

	tc := requireToolCall(t, resp)
	if tc.Function.Name != "go_inspect" {
		t.Fatalf("expected go_inspect, got %s", tc.Function.Name)
	}

	// Actually execute the inspect
	r := inspectFile(t, serverFile)
	if r.Package != "server" {
		t.Errorf("package: got %q, want server", r.Package)
	}
	foundServer := false
	for _, ty := range r.Types {
		if ty.Name == "Server" {
			foundServer = true
		}
	}
	if !foundServer {
		t.Error("Server type not found in inspect result")
	}
	t.Logf("Inspect OK: package=%s, %d types, %d functions", r.Package, len(r.Types), len(r.Functions))
}

// TestLlamaCreateFunctionE2E: model creates a function, we execute the edit tool,
// verify the resulting Go code compiles and contains the function.
func TestLlamaCreateFunctionE2E(t *testing.T) {
	if !llamaAvailable() {
		t.Skip("llama.cpp server not available at " + llamaServerURL)
	}
	dir := scaffold(t)
	configFile := filepath.Join(dir, "internal/config/config.go")

	resp := sendChat(t, chatRequest{
		Model: gemmaE2BModel,
		Messages: []chatMessage{
			{Role: "system", Content: goAstSystemPrompt()},
			{Role: "user", Content: "Use go_create_function to create a new function called ParseConfig that takes path as a string parameter and returns *Config and error in " + configFile},
		},
		Tools:       goAstTools(),
		Temperature: 0.1,
		MaxTokens:   512,
	})

	// Find the edit tool call (model may inspect first)
	var editCall *toolCall
	for _, choice := range resp.Choices {
		for i, tc := range choice.Message.ToolCalls {
			t.Logf("Tool call %d: %s args=%s", i, tc.Function.Name, tc.Function.Arguments)
			if isGoEditTool(tc.Function.Name) {
				editCall = &choice.Message.ToolCalls[i]
			}
		}
	}

	if editCall == nil {
		// Model may have called inspect first — that's fine, give it the result
		tc := requireToolCall(t, resp)
		if tc.Function.Name == "go_inspect" {
			r := inspectFile(t, configFile)
			inspectJSON, _ := json.Marshal(r)

			resp2 := sendChat(t, chatRequest{
				Model: gemmaE2BModel,
				Messages: []chatMessage{
					{Role: "system", Content: goAstSystemPrompt()},
					{Role: "user", Content: "Use go_create_function to create a new function called ParseConfig that takes path as a string parameter and returns *Config and error in " + configFile},
					{Role: "assistant", Content: "", ToolCalls: resp.Choices[0].Message.ToolCalls},
					{Role: "tool", Content: string(inspectJSON), ToolCallID: tc.ID},
				},
				Tools:       goAstTools(),
				Temperature: 0.1,
				MaxTokens:   512,
			})
			for _, choice := range resp2.Choices {
				for i, tc2 := range choice.Message.ToolCalls {
					t.Logf("Turn 2 tool call %d: %s args=%s", i, tc2.Function.Name, tc2.Function.Arguments)
					if isGoEditTool(tc2.Function.Name) {
						editCall = &choice.Message.ToolCalls[i]
					}
				}
			}
		}
	}

	if editCall == nil {
		t.Fatal("model never called an edit tool")
	}

	// Execute the model's edit tool call
	result := executeModelToolCall(t, *editCall, configFile)
	if result.Content == "" {
		t.Fatal("no content returned from edit")
	}

	// Verify the Go code is valid
	content, _ := os.ReadFile(configFile)
	errs := parseErrors(configFile, content)
	if len(errs) > 0 {
		t.Errorf("resulting Go code has parse errors: %v", errs)
	}

	// Verify ParseConfig exists
	r := inspectFile(t, configFile)
	found := false
	for _, f := range r.Functions {
		if f.Name == "ParseConfig" {
			found = true
			t.Logf("ParseConfig signature: %s", f.Signature)
		}
	}
	if !found {
		t.Error("ParseConfig function not found after edit")
		t.Logf("File content:\n%s", string(content))
	}
}

// TestLlamaAddStructFieldE2E: model adds a struct field, we execute, verify.
func TestLlamaAddStructFieldE2E(t *testing.T) {
	if !llamaAvailable() {
		t.Skip("llama.cpp server not available at " + llamaServerURL)
	}
	dir := scaffold(t)
	configFile := filepath.Join(dir, "internal/config/config.go")

	resp := sendChat(t, chatRequest{
		Model: gemmaE2BModel,
		Messages: []chatMessage{
			{Role: "system", Content: goAstSystemPrompt()},
			{Role: "user", Content: "Use go_add_struct_field to add a struct field called Debug of type bool with tag `json:\"debug\"` to the Config struct in " + configFile},
		},
		Tools:       goAstTools(),
		Temperature: 0.1,
		MaxTokens:   512,
	})

	// Find edit tool call
	var editCall *toolCall
	for _, choice := range resp.Choices {
		for i, tc := range choice.Message.ToolCalls {
			t.Logf("Tool: %s args=%s", tc.Function.Name, tc.Function.Arguments)
			if isGoEditTool(tc.Function.Name) {
				editCall = &choice.Message.ToolCalls[i]
			}
		}
	}

	if editCall == nil {
		// Try multi-turn with inspect
		tc := requireToolCall(t, resp)
		if tc.Function.Name == "go_inspect" {
			r := inspectFile(t, configFile)
			inspectJSON, _ := json.Marshal(r)
			resp2 := sendChat(t, chatRequest{
				Model: gemmaE2BModel,
				Messages: []chatMessage{
					{Role: "system", Content: goAstSystemPrompt()},
					{Role: "user", Content: "Use go_add_struct_field to add a struct field called Debug of type bool with tag `json:\"debug\"` to the Config struct in " + configFile},
					{Role: "assistant", Content: "", ToolCalls: resp.Choices[0].Message.ToolCalls},
					{Role: "tool", Content: string(inspectJSON), ToolCallID: tc.ID},
				},
				Tools:       goAstTools(),
				Temperature: 0.1,
				MaxTokens:   512,
			})
			for _, choice := range resp2.Choices {
				for i, tc2 := range choice.Message.ToolCalls {
					t.Logf("Turn 2: %s args=%s", tc2.Function.Name, tc2.Function.Arguments)
					if isGoEditTool(tc2.Function.Name) {
						editCall = &choice.Message.ToolCalls[i]
					}
				}
			}
		}
	}

	if editCall == nil {
		t.Fatal("model never called an edit tool")
	}

	// Execute
	result := executeModelToolCall(t, *editCall, configFile)
	if result.Content == "" {
		t.Fatal("no content from edit")
	}

	// Verify valid Go
	content, _ := os.ReadFile(configFile)
	errs := parseErrors(configFile, content)
	if len(errs) > 0 {
		t.Errorf("parse errors: %v", errs)
	}

	// Verify Debug field exists
	r := inspectFile(t, configFile)
	var cfg *TypeInfo
	for i := range r.Types {
		if r.Types[i].Name == "Config" {
			cfg = &r.Types[i]
		}
	}
	if cfg == nil {
		t.Fatal("Config type not found")
	}
	found := false
	for _, f := range cfg.Fields {
		if f.Name == "Debug" {
			found = true
			if f.Type != "bool" {
				t.Errorf("Debug type: got %q, want bool", f.Type)
			}
			t.Logf("Debug field: type=%s tag=%s", f.Type, f.Tag)
		}
	}
	if !found {
		t.Error("Debug field not found in Config struct")
		t.Logf("Fields: %+v", cfg.Fields)
	}
}

// TestLlamaCreateMethodE2E: the hardest test — model creates a method with
// receiver. We execute the edit tool call and verify the code.
func TestLlamaCreateMethodE2E(t *testing.T) {
	if !llamaAvailable() {
		t.Skip("llama.cpp server not available at " + llamaServerURL)
	}
	dir := scaffold(t)
	serverFile := filepath.Join(dir, "internal/server/server.go")

	// Give the model the inspect result directly to skip a turn
	ir := inspectFile(t, serverFile)
	inspectJSON, _ := json.Marshal(ir)

	resp := sendChat(t, chatRequest{
		Model: gemmaE2BModel,
		Messages: []chatMessage{
			{Role: "system", Content: goAstSystemPrompt()},
			{Role: "user", Content: "I already inspected " + serverFile + ". Here is the result:\n" + string(inspectJSON) + "\n\nNow use go_create_method to create a method called Shutdown on *Server that takes ctx context.Context and returns error."},
		},
		Tools:       goAstTools(),
		Temperature: 0.1,
		MaxTokens:   512,
	})

	tc := requireToolCall(t, resp)
	t.Logf("Tool: %s args=%s", tc.Function.Name, tc.Function.Arguments)

	if !isGoEditTool(tc.Function.Name) {
		t.Fatalf("expected an edit tool, got %s", tc.Function.Name)
	}

	// Parse and log args for diagnostics
	var args map[string]interface{}
	json.Unmarshal([]byte(tc.Function.Arguments), &args)
	t.Logf("Raw args: name=%v receiverType=%v receiverVar=%v",
		args["name"], args["receiverType"], args["receiverVar"])

	// Execute (normalizeOp handles parameter confusion)
	result := executeModelToolCall(t, tc, serverFile)
	if result.Content == "" {
		t.Fatal("no content from edit")
	}

	// Verify valid Go
	content, _ := os.ReadFile(serverFile)
	errs := parseErrors(serverFile, content)
	if len(errs) > 0 {
		t.Errorf("parse errors: %v\nfile:\n%s", errs, string(content))
	}

	// Verify Shutdown method exists
	r := inspectFile(t, serverFile)
	found := false
	for _, f := range r.Functions {
		if f.Name == "Server.Shutdown" || strings.HasSuffix(f.Name, ".Shutdown") {
			found = true
			t.Logf("Found: %s receiver=%s", f.Signature, f.Receiver)
		}
	}
	if !found {
		t.Error("Server.Shutdown method not found")
		t.Logf("Functions: %+v", r.Functions)
		t.Logf("File:\n%s", string(content))
	}
}

// TestLlamaMultiTurnE2E: full inspect → edit → verify cycle with real files.
func TestLlamaMultiTurnE2E(t *testing.T) {
	if !llamaAvailable() {
		t.Skip("llama.cpp server not available at " + llamaServerURL)
	}
	dir := scaffold(t)
	authFile := filepath.Join(dir, "internal/auth/auth.go")

	// Turn 1: Ask model to add import — it should inspect first
	resp1 := sendChat(t, chatRequest{
		Model: gemmaE2BModel,
		Messages: []chatMessage{
			{Role: "system", Content: goAstSystemPrompt()},
			{Role: "user", Content: "Add the import \"context\" to " + authFile + ". First inspect the file, then add the import."},
		},
		Tools:       goAstTools(),
		Temperature: 0.1,
		MaxTokens:   512,
	})

	tc1 := requireToolCall(t, resp1)
	t.Logf("Turn 1: %s(%s)", tc1.Function.Name, tc1.Function.Arguments)

	// Execute whatever the model asked for
	var turn2Content string
	if tc1.Function.Name == "go_inspect" {
		r := inspectFile(t, authFile)
		j, _ := json.Marshal(r)
		turn2Content = string(j)
	} else if isGoEditTool(tc1.Function.Name) {
		result := executeModelToolCall(t, tc1, authFile)
		turn2Content = result.Diff
		if turn2Content == "" {
			turn2Content = "Edit applied successfully."
		}
	}

	// Turn 2: Send result back
	resp2 := sendChat(t, chatRequest{
		Model: gemmaE2BModel,
		Messages: []chatMessage{
			{Role: "system", Content: goAstSystemPrompt()},
			{Role: "user", Content: "Add the import \"context\" to " + authFile + ". First inspect the file, then add the import."},
			{Role: "assistant", Content: "", ToolCalls: resp1.Choices[0].Message.ToolCalls},
			{Role: "tool", Content: turn2Content, ToolCallID: tc1.ID},
		},
		Tools:       goAstTools(),
		Temperature: 0.1,
		MaxTokens:   512,
	})

	if len(resp2.Choices) > 0 {
		msg2 := resp2.Choices[0].Message
		for _, tc2 := range msg2.ToolCalls {
			t.Logf("Turn 2: %s(%s)", tc2.Function.Name, tc2.Function.Arguments)
			if isGoEditTool(tc2.Function.Name) {
				executeModelToolCall(t, tc2, authFile)
			}
		}
	}

	// Verify: context import should exist
	r := inspectFile(t, authFile)
	found := false
	for _, imp := range r.Imports {
		if imp.Path == "context" {
			found = true
		}
	}
	if !found {
		t.Error("context import not found after multi-turn edit")
		content, _ := os.ReadFile(authFile)
		t.Logf("File:\n%s", string(content))
	}

	// Verify valid Go
	content, _ := os.ReadFile(authFile)
	errs := parseErrors(authFile, content)
	if len(errs) > 0 {
		t.Errorf("parse errors: %v", errs)
	}
}

// TestLlamaUsesASTToolsOnly verifies the model uses go_* AST tools (not edit/write)
// when asked to modify a Go file, even with text-based tools available.
func TestLlamaUsesASTToolsOnly(t *testing.T) {
	if !llamaAvailable() {
		t.Skip("llama.cpp server not available at " + llamaServerURL)
	}

	allTools := append(goAstTools(), blockedTools()...)

	resp := sendChat(t, chatRequest{
		Model: gemmaE2BModel,
		Messages: []chatMessage{
			{Role: "system", Content: goAstSystemPrompt()},
			{Role: "user", Content: "Add a new struct field 'Debug bool' with json tag to the Config struct in /tmp/project/config.go"},
		},
		Tools:       allTools,
		Temperature: 0.1,
		MaxTokens:   512,
	})

	if len(resp.Choices) == 0 {
		t.Fatal("no choices")
	}
	msg := resp.Choices[0].Message
	if len(msg.ToolCalls) == 0 {
		t.Fatalf("expected tool call, got: %s", msg.Content)
	}
	for _, tc := range msg.ToolCalls {
		t.Logf("Model chose: %s", tc.Function.Name)
		if tc.Function.Name == "edit" || tc.Function.Name == "write" {
			t.Errorf("model used blocked tool %q instead of Go AST tools", tc.Function.Name)
		}
	}
}

// TestLlamaReasoningContent verifies the server returns reasoning_content.
func TestLlamaReasoningContent(t *testing.T) {
	if !llamaAvailable() {
		t.Skip("llama.cpp server not available at " + llamaServerURL)
	}

	resp := sendChat(t, chatRequest{
		Model: gemmaE2BModel,
		Messages: []chatMessage{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "What is 2+2?"},
		},
		Temperature: 0.1,
		MaxTokens:   128,
	})

	if len(resp.Choices) == 0 {
		t.Fatal("no choices")
	}
	msg := resp.Choices[0].Message
	if msg.ReasoningContent == "" {
		t.Log("Warning: no reasoning_content — model may not have thinking enabled")
	} else {
		t.Logf("Reasoning (%d chars): %s", len(msg.ReasoningContent), truncate(msg.ReasoningContent, 200))
	}
	if msg.Content == "" {
		t.Error("no content in response")
	}
}

// TestLlamaServerFingerprint verifies the server version.
func TestLlamaServerFingerprint(t *testing.T) {
	if !llamaAvailable() {
		t.Skip("llama.cpp server not available at " + llamaServerURL)
	}

	resp := sendChat(t, chatRequest{
		Model: gemmaE2BModel,
		Messages: []chatMessage{
			{Role: "user", Content: "hi"},
		},
		Temperature: 0.1,
		MaxTokens:   16,
	})

	t.Logf("Server fingerprint: %s", resp.SystemFingerprint)
	t.Logf("Model: %s", resp.Model)
}

// ============================================================================
// Part 3: Complex Natural Language Refactoring
// ============================================================================

// scaffoldRawSQLApp creates a web app with raw SQL queries in HTTP handlers.
func scaffoldRawSQLApp(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	files := map[string]string{
		"go.mod": "module github.com/example/webapp\n\ngo 1.22\n",
		"main.go": `package main

import (
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/users", HandleListUsers)
	http.HandleFunc("/user", HandleGetUser)
	http.HandleFunc("/user/create", HandleCreateUser)
	http.HandleFunc("/user/delete", HandleDeleteUser)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
`,
		"handlers.go": `package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

var db *sql.DB

type User struct {
	ID    int    ` + "`" + `json:"id"` + "`" + `
	Name  string ` + "`" + `json:"name"` + "`" + `
	Email string ` + "`" + `json:"email"` + "`" + `
}

func HandleListUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, email FROM users ORDER BY name")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		var u User
		rows.Scan(&u.ID, &u.Name, &u.Email)
		users = append(users, u)
	}
	json.NewEncoder(w).Encode(users)
}

func HandleGetUser(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	var u User
	err := db.QueryRow("SELECT id, name, email FROM users WHERE id = ?", id).Scan(&u.ID, &u.Name, &u.Email)
	if err != nil {
		http.Error(w, "not found", 404)
		return
	}
	json.NewEncoder(w).Encode(u)
}

func HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	var u User
	json.NewDecoder(r.Body).Decode(&u)
	result, err := db.Exec("INSERT INTO users (name, email) VALUES (?, ?)", u.Name, u.Email)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	id, _ := result.LastInsertId()
	u.ID = int(id)
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(u)
}

func HandleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	_, err := db.Exec("DELETE FROM users WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(204)
}
`,
	}

	for name, content := range files {
		p := filepath.Join(dir, name)
		os.MkdirAll(filepath.Dir(p), 0755)
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

// TestLlamaComplexRefactorMVC is a full end-to-end test of complex natural
// language refactoring. The model receives a high-level English prompt and
// must decompose it into a sequence of go_* tool calls that produce valid
// Go code. We run a multi-turn agent loop, executing every tool call through
// the Go AST helper and feeding results back.
func TestLlamaComplexRefactorMVC(t *testing.T) {
	if !llamaAvailable() {
		t.Skip("llama.cpp server not available at " + llamaServerURL)
	}

	dir := scaffoldRawSQLApp(t)
	handlersFile := filepath.Join(dir, "handlers.go")

	// Inspect handlers.go to provide context (mirrors OpenCode's file context)
	ir := inspectFile(t, handlersFile)
	inspectJSON, _ := json.Marshal(ir)

	// Natural language prompt — the model must decompose this into individual tool calls
	prompt := "I have a Go web application. Here is the AST structure of " + handlersFile + ":\n\n" +
		string(inspectJSON) + "\n\n" +
		"Replace the raw SQL queries in this web framework with an MVC approach:\n" +
		"1. Create a UserRepository struct with a db field of type *sql.DB\n" +
		"2. Add methods to UserRepository for the database operations (e.g. ListAll, GetByID, Create, Delete)\n" +
		"3. Each repository method should have the appropriate parameters and return types\n\n" +
		"Use the Go AST editing tools to make these changes to " + handlersFile + ". Work step by step, one operation per tool call."

	messages := []chatMessage{
		{Role: "system", Content: goAstSystemPrompt()},
		{Role: "user", Content: prompt},
	}

	var ops []string
	var successEdits, failedEdits int
	consecutiveFails := 0
	maxTurns := 20

	for turn := 0; turn < maxTurns; turn++ {
		resp := sendChat(t, chatRequest{
			Model:       gemmaE2BModel,
			Messages:    messages,
			Tools:       goAstTools(),
			Temperature: 0.1,
			MaxTokens:   1024,
		})

		if len(resp.Choices) == 0 {
			t.Logf("Turn %d: no choices returned", turn+1)
			break
		}
		msg := resp.Choices[0].Message

		if len(msg.ToolCalls) == 0 {
			// Model is done — returned text instead of tool calls
			t.Logf("Turn %d: model finished — %s", turn+1, truncate(msg.Content, 300))
			break
		}

		// Add assistant message with tool calls to history.
		// Sanitize tool call arguments — if the model emitted malformed JSON,
		// replace with valid empty JSON so the server doesn't choke on the
		// next request when it re-serializes the conversation history.
		sanitizedCalls := make([]toolCall, len(msg.ToolCalls))
		for i, tc := range msg.ToolCalls {
			sanitizedCalls[i] = tc
			var check json.RawMessage
			if json.Unmarshal([]byte(tc.Function.Arguments), &check) != nil {
				sanitizedCalls[i].Function.Arguments = `{"error":"malformed"}`
			}
		}
		messages = append(messages, chatMessage{
			Role:      "assistant",
			ToolCalls: sanitizedCalls,
		})

		turnHadSuccess := false

		// Execute each tool call
		for _, tc := range msg.ToolCalls {
			t.Logf("Turn %d: %s(%s)", turn+1, tc.Function.Name, truncate(tc.Function.Arguments, 400))

			var result string

			// Dispatch: any tool starting with "go_" that isn't go_inspect/go_fix
			// is an edit operation. The tool name encodes the op directly.
			switch {
			case tc.Function.Name == "go_inspect":
				var args map[string]interface{}
				json.Unmarshal([]byte(tc.Function.Arguments), &args)
				fp, _ := args["filePath"].(string)
				if fp == "" {
					fp = handlersFile
				}
				r, err := inspect(fp)
				if err != nil {
					errB, _ := json.Marshal(map[string]string{"error": err.Error()})
					result = string(errB)
				} else {
					j, _ := json.Marshal(r)
					result = string(j)
				}
				ops = append(ops, "inspect")
				turnHadSuccess = true

			case tc.Function.Name == "go_fix":
				var args map[string]interface{}
				json.Unmarshal([]byte(tc.Function.Arguments), &args)
				fp, _ := args["filePath"].(string)
				if fp == "" {
					fp = handlersFile
				}
				r, err := fix(fp)
				if err != nil {
					errB, _ := json.Marshal(map[string]string{"error": err.Error()})
					result = string(errB)
				} else {
					j, _ := json.Marshal(r)
					result = string(j)
				}
				ops = append(ops, "fix")
				turnHadSuccess = true

			case strings.HasPrefix(tc.Function.Name, "go_"):
				// Edit operation — tool name gives the op
				// Pre-validate JSON — model sometimes emits malformed args
				var rawCheck map[string]interface{}
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &rawCheck); err != nil {
					errB, _ := json.Marshal(map[string]string{
						"error": "malformed tool call arguments: " + err.Error(),
						"hint":  "Arguments must be valid JSON. Do not mix tool call tokens with arguments.",
					})
					result = string(errB)
					ops = append(ops, "FAIL:parse:"+err.Error())
					failedEdits++
					break
				}
				op := parseToolCallOp(t, tc, "")
				if op.File == "" {
					op.File = handlersFile
				}
				r, err := edit(op)
				if err != nil {
					errB, _ := json.Marshal(map[string]string{
						"error": err.Error(),
						"hint":  "Make sure filePath is set and the target exists. For go_create_struct use name (not target).",
					})
					result = string(errB)
					ops = append(ops, "FAIL:"+op.Op+":"+err.Error())
					failedEdits++
				} else {
					if r.Content != "" {
						os.WriteFile(op.File, []byte(r.Content), 0644)
					}
					result = r.Diff
					if result == "" {
						result = "No changes."
					}
					ops = append(ops, "OK:"+op.Op+":"+op.Target+":"+op.Name)
					successEdits++
					turnHadSuccess = true
				}

			default:
				result = `{"error":"unknown tool"}`
			}

			// Truncate large results to prevent context overflow
			if len(result) > 3000 {
				result = result[:3000] + "\n...(truncated)"
			}

			messages = append(messages, chatMessage{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			})
		}

		// Break if model is stuck in a loop of failures
		if turnHadSuccess {
			consecutiveFails = 0
		} else {
			consecutiveFails++
			if consecutiveFails >= 3 {
				t.Logf("Turn %d: breaking — %d consecutive failed turns", turn+1, consecutiveFails)
				break
			}
		}
	}

	// ---- Verification ----

	t.Logf("=== Operations performed (%d total, %d ok, %d failed) ===", len(ops), successEdits, failedEdits)
	for i, op := range ops {
		t.Logf("  [%d] %s", i+1, op)
	}

	// 1. Model must have made at least one successful edit
	if successEdits == 0 {
		t.Fatal("model made no successful edit operations — refactoring did not start")
	}

	// 2. handlers.go must still be valid Go
	content, err := os.ReadFile(handlersFile)
	if err != nil {
		t.Fatalf("read handlers.go: %v", err)
	}
	errs := parseErrors(handlersFile, content)
	if len(errs) > 0 {
		t.Errorf("handlers.go has parse errors after refactoring:\n%v", errs)
		t.Logf("File content:\n%s", string(content))
	}

	// 3. main.go must still be valid Go (should be untouched)
	mainContent, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	mainErrs := parseErrors(filepath.Join(dir, "main.go"), mainContent)
	if len(mainErrs) > 0 {
		t.Errorf("main.go has parse errors: %v", mainErrs)
	}

	// 4. Structural verification — inspect the final state
	finalInspect := inspectFile(t, handlersFile)
	t.Logf("=== Final file structure ===")
	t.Logf("Types (%d):", len(finalInspect.Types))
	for _, ty := range finalInspect.Types {
		t.Logf("  %s (%s) — %d fields", ty.Name, ty.Kind, len(ty.Fields))
		for _, f := range ty.Fields {
			t.Logf("    .%s %s", f.Name, f.Type)
		}
	}
	t.Logf("Functions/Methods (%d):", len(finalInspect.Functions))
	for _, fn := range finalInspect.Functions {
		t.Logf("  %s", fn.Signature)
	}

	// Check for new types (beyond the original User struct)
	newTypes := 0
	var repoTypeName string
	for _, ty := range finalInspect.Types {
		if ty.Name != "User" {
			newTypes++
			repoTypeName = ty.Name
			t.Logf("New type created: %s (%s)", ty.Name, ty.Kind)
		}
	}

	// Check for new methods (not the original Handle* functions)
	newMethods := 0
	for _, fn := range finalInspect.Functions {
		if !strings.HasPrefix(fn.Name, "Handle") && strings.Contains(fn.Name, ".") {
			newMethods++
		}
	}

	t.Logf("=== Summary: %d new types, %d new methods, %d successful edits, %d failed ===",
		newTypes, newMethods, successEdits, failedEdits)

	// The model must have created at least one new type (the repository)
	if newTypes == 0 {
		t.Error("no new types created — expected at least a repository struct for MVC refactoring")
	} else {
		t.Logf("MVC refactoring created repository type: %s", repoTypeName)
	}

	// If a repository type exists, it should have at least one method
	if newTypes > 0 && newMethods == 0 {
		t.Log("Warning: repository type created but no methods added yet")
	}
	if newMethods > 0 {
		t.Logf("Repository has %d methods — MVC pattern successfully applied", newMethods)
	}
}
