package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func testdataPath(name string) string {
	_, f, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(f), "testdata", name)
}

func TestInspectBasic(t *testing.T) {
	result, err := inspect(testdataPath("basic.go"))
	if err != nil {
		t.Fatalf("inspect failed: %v", err)
	}

	if result.Package != "basic" {
		t.Errorf("expected package basic, got %s", result.Package)
	}

	// Imports
	if len(result.Imports) != 3 {
		t.Errorf("expected 3 imports, got %d", len(result.Imports))
	}

	// Types
	typeNames := map[string]bool{}
	for _, ti := range result.Types {
		typeNames[ti.Name] = true
	}
	for _, name := range []string{"Config", "Handler", "Logger", "Duration", "HandlerFunc", "Server"} {
		if !typeNames[name] {
			t.Errorf("expected type %s", name)
		}
	}

	// Config struct fields
	var configType *TypeInfo
	for i := range result.Types {
		if result.Types[i].Name == "Config" {
			configType = &result.Types[i]
			break
		}
	}
	if configType == nil {
		t.Fatal("Config type not found")
	}
	if configType.Kind != "struct" {
		t.Errorf("expected kind struct, got %s", configType.Kind)
	}
	if len(configType.Fields) != 4 {
		t.Errorf("expected 4 fields, got %d", len(configType.Fields))
	}
	// Check Port field has tag
	if configType.Fields[0].Name != "Port" {
		t.Errorf("expected first field Port, got %s", configType.Fields[0].Name)
	}
	if configType.Fields[0].Tag == "" {
		t.Error("expected Port field to have tag")
	}

	// Handler interface
	var handlerType *TypeInfo
	for i := range result.Types {
		if result.Types[i].Name == "Handler" {
			handlerType = &result.Types[i]
			break
		}
	}
	if handlerType == nil {
		t.Fatal("Handler type not found")
	}
	if handlerType.Kind != "interface" {
		t.Errorf("expected kind interface, got %s", handlerType.Kind)
	}
	if len(handlerType.Methods) != 2 {
		t.Errorf("expected 2 methods, got %d", len(handlerType.Methods))
	}

	// Functions
	funcNames := map[string]bool{}
	for _, fi := range result.Functions {
		funcNames[fi.Name] = true
	}
	for _, name := range []string{"NewServer", "Server.Start", "Server.Stop", "HandleRequest", "init"} {
		if !funcNames[name] {
			t.Errorf("expected function %s", name)
		}
	}

	// Server.Start method details
	var startFunc *FunctionInfo
	for i := range result.Functions {
		if result.Functions[i].Name == "Server.Start" {
			startFunc = &result.Functions[i]
			break
		}
	}
	if startFunc == nil {
		t.Fatal("Server.Start not found")
	}
	if startFunc.Receiver != "*Server" {
		t.Errorf("expected receiver *Server, got %s", startFunc.Receiver)
	}
	if startFunc.ReceiverVar != "s" {
		t.Errorf("expected receiverVar s, got %s", startFunc.ReceiverVar)
	}
	if len(startFunc.Params) != 1 {
		t.Errorf("expected 1 param, got %d", len(startFunc.Params))
	}
	if len(startFunc.Returns) != 1 {
		t.Errorf("expected 1 return, got %d", len(startFunc.Returns))
	}
	if len(startFunc.Body) != 3 {
		t.Errorf("expected 3 body statements, got %d", len(startFunc.Body))
	}
	if startFunc.Doc != "Start begins listening for requests." {
		t.Errorf("expected doc comment, got %q", startFunc.Doc)
	}

	// Consts
	constNames := map[string]bool{}
	for _, ci := range result.Consts {
		constNames[ci.Name] = true
	}
	for _, name := range []string{"DefaultPort", "MaxRetries", "MinTimeout"} {
		if !constNames[name] {
			t.Errorf("expected const %s", name)
		}
	}

	// Vars
	varNames := map[string]bool{}
	for _, vi := range result.Vars {
		varNames[vi.Name] = true
	}
	for _, name := range []string{"ErrNotFound", "globalLogger", "globalConfig"} {
		if !varNames[name] {
			t.Errorf("expected var %s", name)
		}
	}
}

func TestInspectImports(t *testing.T) {
	result, err := inspect(testdataPath("imports.go"))
	if err != nil {
		t.Fatalf("inspect failed: %v", err)
	}

	if result.Package != "imports" {
		t.Errorf("expected package imports, got %s", result.Package)
	}

	// Should have 7 imports
	if len(result.Imports) != 7 {
		t.Errorf("expected 7 imports, got %d", len(result.Imports))
	}

	// Check aliased import
	found := false
	for _, imp := range result.Imports {
		if imp.Path == "google.golang.org/protobuf/proto" && imp.Alias == "pb" {
			found = true
		}
	}
	if !found {
		t.Error("expected aliased import pb for google.golang.org/protobuf/proto")
	}

	// Check blank import
	found = false
	for _, imp := range result.Imports {
		if imp.Path == "net/http/pprof" && imp.Alias == "_" {
			found = true
		}
	}
	if !found {
		t.Error("expected blank import for net/http/pprof")
	}
}

func TestInspectComplex(t *testing.T) {
	result, err := inspect(testdataPath("complex.go"))
	if err != nil {
		t.Fatalf("inspect failed: %v", err)
	}

	// Build constraint
	if result.BuildConstraint != "linux && amd64" {
		t.Errorf("expected build constraint 'linux && amd64', got %q", result.BuildConstraint)
	}

	// Generate directive
	if len(result.GenerateDirectives) != 1 {
		t.Errorf("expected 1 generate directive, got %d", len(result.GenerateDirectives))
	}

	// Named type Color
	var colorType *TypeInfo
	for i := range result.Types {
		if result.Types[i].Name == "Color" {
			colorType = &result.Types[i]
			break
		}
	}
	if colorType == nil {
		t.Fatal("Color type not found")
	}
	if colorType.Kind != "named" {
		t.Errorf("expected kind named, got %s", colorType.Kind)
	}

	// Interface with embed
	var procType *TypeInfo
	for i := range result.Types {
		if result.Types[i].Name == "Processor" {
			procType = &result.Types[i]
			break
		}
	}
	if procType == nil {
		t.Fatal("Processor type not found")
	}
	if len(procType.Embeds) != 1 {
		t.Errorf("expected 1 embed, got %d", len(procType.Embeds))
	}
	if procType.Embeds[0] != "io.Closer" {
		t.Errorf("expected embed io.Closer, got %s", procType.Embeds[0])
	}

	// ProcessAll function with variadic params
	var processAll *FunctionInfo
	for i := range result.Functions {
		if result.Functions[i].Name == "ProcessAll" {
			processAll = &result.Functions[i]
			break
		}
	}
	if processAll == nil {
		t.Fatal("ProcessAll not found")
	}
	if len(processAll.Params) != 3 {
		t.Errorf("expected 3 params, got %d", len(processAll.Params))
	}
	if len(processAll.Returns) != 2 {
		t.Errorf("expected 2 returns, got %d", len(processAll.Returns))
	}
}

func TestInspectOccurrence(t *testing.T) {
	// Create a temp file with duplicate declarations
	src := `package test

type Shape struct {
	Name string
}

func Shape() {}

type Shape struct{}
type Shape struct{}
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "dupes.go")
	if err := writeTestFile(tmpFile, src); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	result, err := inspect(tmpFile)
	if err != nil {
		t.Fatalf("inspect failed: %v", err)
	}

	// Should have 3 types named "Shape" with occurrences
	shapeTypes := 0
	for _, ti := range result.Types {
		if ti.Name == "Shape" {
			shapeTypes++
		}
	}
	if shapeTypes != 3 {
		t.Errorf("expected 3 Shape types, got %d", shapeTypes)
	}

	// Check occurrence numbers (global file order: type#1, func#2, type#3, type#4)
	// Types array: Shape(occ 1), Shape(occ 3), Shape(occ 4)
	// Functions array: Shape(occ 2)
	typeOccs := []int{}
	for _, ti := range result.Types {
		if ti.Name == "Shape" {
			typeOccs = append(typeOccs, ti.Occurrence)
		}
	}
	if len(typeOccs) != 3 || typeOccs[0] != 1 || typeOccs[1] != 3 || typeOccs[2] != 4 {
		t.Errorf("expected type occurrences [1,3,4], got %v", typeOccs)
	}

	funcOccs := []int{}
	for _, fi := range result.Functions {
		if fi.Name == "Shape" {
			funcOccs = append(funcOccs, fi.Occurrence)
		}
	}
	if len(funcOccs) != 1 || funcOccs[0] != 2 {
		t.Errorf("expected func occurrence [2], got %v", funcOccs)
	}

	// No duplicates → occurrence should be 0 (omitted)
	src2 := `package test
type Foo struct{}
func Bar() {}
`
	tmpFile2 := filepath.Join(tmpDir, "nodupes.go")
	if err := writeTestFile(tmpFile2, src2); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	result2, err := inspect(tmpFile2)
	if err != nil {
		t.Fatalf("inspect failed: %v", err)
	}
	for _, ti := range result2.Types {
		if ti.Occurrence != 0 {
			t.Errorf("expected no occurrence for unique type %s, got %d", ti.Name, ti.Occurrence)
		}
	}
	for _, fi := range result2.Functions {
		if fi.Occurrence != 0 {
			t.Errorf("expected no occurrence for unique func %s, got %d", fi.Name, fi.Occurrence)
		}
	}
}

func writeTestFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func TestInspectEmpty(t *testing.T) {
	result, err := inspect(testdataPath("empty.go"))
	if err != nil {
		t.Fatalf("inspect failed: %v", err)
	}

	if result.Package != "empty" {
		t.Errorf("expected package empty, got %s", result.Package)
	}
	if len(result.Imports) != 0 {
		t.Errorf("expected 0 imports, got %d", len(result.Imports))
	}
	if len(result.Types) != 0 {
		t.Errorf("expected 0 types, got %d", len(result.Types))
	}
	if len(result.Functions) != 0 {
		t.Errorf("expected 0 functions, got %d", len(result.Functions))
	}
}
