package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func parseTestFile(t *testing.T, src string) (*ast.File, *token.FileSet) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("failed to parse test source: %v", err)
	}
	return file, fset
}

func TestResolveFunction(t *testing.T) {
	src := `package test
func Foo() {}
func Bar(x int) string { return "" }
`
	file, fset := parseTestFile(t, src)

	r, err := ResolveTarget(file, fset, "Foo")
	if err != nil {
		t.Fatalf("resolve Foo: %v", err)
	}
	fd, ok := r.Node.(*ast.FuncDecl)
	if !ok {
		t.Fatalf("expected *ast.FuncDecl, got %T", r.Node)
	}
	if fd.Name.Name != "Foo" {
		t.Errorf("expected name Foo, got %s", fd.Name.Name)
	}

	r, err = ResolveTarget(file, fset, "Bar")
	if err != nil {
		t.Fatalf("resolve Bar: %v", err)
	}
	fd, ok = r.Node.(*ast.FuncDecl)
	if !ok {
		t.Fatalf("expected *ast.FuncDecl, got %T", r.Node)
	}
	if fd.Name.Name != "Bar" {
		t.Errorf("expected name Bar, got %s", fd.Name.Name)
	}
}

func TestResolveMethod(t *testing.T) {
	src := `package test
type Server struct{}
func (s *Server) Start() error { return nil }
func (s Server) Name() string { return "" }
`
	file, fset := parseTestFile(t, src)

	r, err := ResolveTarget(file, fset, "Server.Start")
	if err != nil {
		t.Fatalf("resolve Server.Start: %v", err)
	}
	fd, ok := r.Node.(*ast.FuncDecl)
	if !ok {
		t.Fatalf("expected *ast.FuncDecl, got %T", r.Node)
	}
	if fd.Name.Name != "Start" {
		t.Errorf("expected name Start, got %s", fd.Name.Name)
	}

	r, err = ResolveTarget(file, fset, "Server.Name")
	if err != nil {
		t.Fatalf("resolve Server.Name: %v", err)
	}
	fd, ok = r.Node.(*ast.FuncDecl)
	if !ok {
		t.Fatalf("expected *ast.FuncDecl, got %T", r.Node)
	}
	if fd.Name.Name != "Name" {
		t.Errorf("expected name Name, got %s", fd.Name.Name)
	}
}

func TestResolveType(t *testing.T) {
	src := `package test
type Config struct {
	Port int
}
type Handler interface {
	Handle()
}
`
	file, fset := parseTestFile(t, src)

	r, err := ResolveTarget(file, fset, "Config")
	if err != nil {
		t.Fatalf("resolve Config: %v", err)
	}
	ts, ok := r.Node.(*ast.TypeSpec)
	if !ok {
		t.Fatalf("expected *ast.TypeSpec, got %T", r.Node)
	}
	if ts.Name.Name != "Config" {
		t.Errorf("expected name Config, got %s", ts.Name.Name)
	}

	r, err = ResolveTarget(file, fset, "Handler")
	if err != nil {
		t.Fatalf("resolve Handler: %v", err)
	}
	ts, ok = r.Node.(*ast.TypeSpec)
	if !ok {
		t.Fatalf("expected *ast.TypeSpec, got %T", r.Node)
	}
	if ts.Name.Name != "Handler" {
		t.Errorf("expected name Handler, got %s", ts.Name.Name)
	}
}

func TestResolveStructField(t *testing.T) {
	src := `package test
type Config struct {
	Port int
	Host string
}
`
	file, fset := parseTestFile(t, src)

	r, err := ResolveTarget(file, fset, "Config.Port")
	if err != nil {
		t.Fatalf("resolve Config.Port: %v", err)
	}
	f, ok := r.Node.(*ast.Field)
	if !ok {
		t.Fatalf("expected *ast.Field, got %T", r.Node)
	}
	if f.Names[0].Name != "Port" {
		t.Errorf("expected name Port, got %s", f.Names[0].Name)
	}

	r, err = ResolveTarget(file, fset, "Config.Host")
	if err != nil {
		t.Fatalf("resolve Config.Host: %v", err)
	}
	f, ok = r.Node.(*ast.Field)
	if !ok {
		t.Fatalf("expected *ast.Field, got %T", r.Node)
	}
	if f.Names[0].Name != "Host" {
		t.Errorf("expected name Host, got %s", f.Names[0].Name)
	}
}

func TestResolveInterfaceMethod(t *testing.T) {
	src := `package test
type Handler interface {
	ServeHTTP(w int, r int)
	Close() error
}
`
	file, fset := parseTestFile(t, src)

	r, err := ResolveTarget(file, fset, "Handler.ServeHTTP")
	if err != nil {
		t.Fatalf("resolve Handler.ServeHTTP: %v", err)
	}
	f, ok := r.Node.(*ast.Field)
	if !ok {
		t.Fatalf("expected *ast.Field, got %T", r.Node)
	}
	if f.Names[0].Name != "ServeHTTP" {
		t.Errorf("expected name ServeHTTP, got %s", f.Names[0].Name)
	}
}

func TestResolveVarConst(t *testing.T) {
	src := `package test
const MaxRetries = 3
var ErrNotFound = "not found"
`
	file, fset := parseTestFile(t, src)

	r, err := ResolveTarget(file, fset, "MaxRetries")
	if err != nil {
		t.Fatalf("resolve MaxRetries: %v", err)
	}
	vs, ok := r.Node.(*ast.ValueSpec)
	if !ok {
		t.Fatalf("expected *ast.ValueSpec, got %T", r.Node)
	}
	if vs.Names[0].Name != "MaxRetries" {
		t.Errorf("expected name MaxRetries, got %s", vs.Names[0].Name)
	}

	r, err = ResolveTarget(file, fset, "ErrNotFound")
	if err != nil {
		t.Fatalf("resolve ErrNotFound: %v", err)
	}
	vs, ok = r.Node.(*ast.ValueSpec)
	if !ok {
		t.Fatalf("expected *ast.ValueSpec, got %T", r.Node)
	}
	if vs.Names[0].Name != "ErrNotFound" {
		t.Errorf("expected name ErrNotFound, got %s", vs.Names[0].Name)
	}
}

func TestResolveFuncParam(t *testing.T) {
	src := `package test
func Handle(ctx int, id string) {}
`
	file, fset := parseTestFile(t, src)

	r, err := ResolveTarget(file, fset, "Handle.ctx")
	if err != nil {
		t.Fatalf("resolve Handle.ctx: %v", err)
	}
	f, ok := r.Node.(*ast.Field)
	if !ok {
		t.Fatalf("expected *ast.Field, got %T", r.Node)
	}
	if f.Names[0].Name != "ctx" {
		t.Errorf("expected name ctx, got %s", f.Names[0].Name)
	}
}

func TestResolveCompoundIfTarget(t *testing.T) {
	src := `package test
func Foo() {
	x := 1
	if x > 0 {
		y := 2
		_ = y
	}
}
`
	file, fset := parseTestFile(t, src)

	r, err := ResolveTarget(file, fset, "Foo.if[1]")
	if err != nil {
		t.Fatalf("resolve Foo.if[1]: %v", err)
	}
	if r.StmtList == nil {
		t.Fatal("expected StmtList to be set")
	}
	if len(*r.StmtList) != 2 {
		t.Errorf("expected 2 statements in if body, got %d", len(*r.StmtList))
	}
}

func TestResolveCompoundForTarget(t *testing.T) {
	src := `package test
func Foo() {
	items := []int{1, 2}
	for _, v := range items {
		_ = v
	}
}
`
	file, fset := parseTestFile(t, src)

	r, err := ResolveTarget(file, fset, "Foo.for[1]")
	if err != nil {
		t.Fatalf("resolve Foo.for[1]: %v", err)
	}
	if r.StmtList == nil {
		t.Fatal("expected StmtList to be set")
	}
	if len(*r.StmtList) != 1 {
		t.Errorf("expected 1 statement in for body, got %d", len(*r.StmtList))
	}
}

func TestResolveCompoundElseTarget(t *testing.T) {
	src := `package test
func Foo() {
	x := 1
	if x > 0 {
		_ = x
	} else {
		y := 2
		_ = y
	}
}
`
	file, fset := parseTestFile(t, src)

	r, err := ResolveTarget(file, fset, "Foo.if[1].else")
	if err != nil {
		t.Fatalf("resolve Foo.if[1].else: %v", err)
	}
	if r.StmtList == nil {
		t.Fatal("expected StmtList to be set")
	}
	if len(*r.StmtList) != 2 {
		t.Errorf("expected 2 statements in else body, got %d", len(*r.StmtList))
	}
}

func TestResolveOccurrence(t *testing.T) {
	src := `package test

type Shape struct {
	Name string
}

type Shape struct{}
type Shape struct{}
`
	file, fset := parseTestFile(t, src)

	// Shape (no #N) targets the first occurrence
	r, err := ResolveTarget(file, fset, "Shape")
	if err != nil {
		t.Fatalf("resolve Shape: %v", err)
	}
	ts, ok := r.Node.(*ast.TypeSpec)
	if !ok {
		t.Fatalf("expected *ast.TypeSpec, got %T", r.Node)
	}
	// First Shape has a field "Name"
	st, ok := ts.Type.(*ast.StructType)
	if !ok || st.Fields == nil || len(st.Fields.List) == 0 {
		t.Fatal("expected first Shape to have fields")
	}

	// Shape#2 targets the second occurrence (empty struct)
	r, err = ResolveTarget(file, fset, "Shape#2")
	if err != nil {
		t.Fatalf("resolve Shape#2: %v", err)
	}
	ts, ok = r.Node.(*ast.TypeSpec)
	if !ok {
		t.Fatalf("expected *ast.TypeSpec, got %T", r.Node)
	}
	st, ok = ts.Type.(*ast.StructType)
	if !ok {
		t.Fatal("expected *ast.StructType")
	}
	if st.Fields != nil && len(st.Fields.List) > 0 {
		t.Error("expected second Shape to have no fields")
	}

	// Shape#3 targets the third occurrence
	r, err = ResolveTarget(file, fset, "Shape#3")
	if err != nil {
		t.Fatalf("resolve Shape#3: %v", err)
	}
	ts, ok = r.Node.(*ast.TypeSpec)
	if !ok {
		t.Fatalf("expected *ast.TypeSpec, got %T", r.Node)
	}

	// Shape#4 does not exist — should get helpful error
	_, err = ResolveTarget(file, fset, "Shape#4")
	if err == nil {
		t.Fatal("expected error for Shape#4")
	}
	if !strings.Contains(err.Error(), "3 declaration(s)") {
		t.Errorf("expected error to mention count, got: %s", err.Error())
	}
}

func TestResolveOccurrenceCrossKind(t *testing.T) {
	// Function and type with same name — occurrence counts in file order
	src := `package test

func Shape() {}
type Shape struct{}
type Shape struct{}
`
	file, fset := parseTestFile(t, src)

	// Shape#1 = the function (first in file)
	r, err := ResolveTarget(file, fset, "Shape")
	if err != nil {
		t.Fatalf("resolve Shape: %v", err)
	}
	if _, ok := r.Node.(*ast.FuncDecl); !ok {
		t.Fatalf("expected first Shape to be FuncDecl, got %T", r.Node)
	}

	// Shape#2 = the first type
	r, err = ResolveTarget(file, fset, "Shape#2")
	if err != nil {
		t.Fatalf("resolve Shape#2: %v", err)
	}
	if _, ok := r.Node.(*ast.TypeSpec); !ok {
		t.Fatalf("expected Shape#2 to be TypeSpec, got %T", r.Node)
	}

	// Shape#3 = the second type
	r, err = ResolveTarget(file, fset, "Shape#3")
	if err != nil {
		t.Fatalf("resolve Shape#3: %v", err)
	}
	if _, ok := r.Node.(*ast.TypeSpec); !ok {
		t.Fatalf("expected Shape#3 to be TypeSpec, got %T", r.Node)
	}
}

func TestResolveNotFound(t *testing.T) {
	src := `package test
func Foo() {}
`
	file, fset := parseTestFile(t, src)

	_, err := ResolveTarget(file, fset, "NonExistent")
	if err == nil {
		t.Fatal("expected error for nonexistent target")
	}

	_, err = ResolveTarget(file, fset, "Foo.badparam")
	if err == nil {
		t.Fatal("expected error for nonexistent sub-target")
	}
}
