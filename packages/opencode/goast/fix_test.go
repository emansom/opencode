package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempGo(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "test.go")
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestFixDoubleSemicolon(t *testing.T) {
	src := `package main

func foo() {
	x := 1;;
	y := 2;;
}
`
	p := writeTempGo(t, src)
	result, err := fix(p)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("expected success, got remaining errors: %v", result.RemainingErrs)
	}
	if len(result.FixesApplied) == 0 {
		t.Fatal("expected fixes to be applied")
	}

	got, _ := os.ReadFile(p)
	if strings.Contains(string(got), ";;") {
		t.Error("double semicolons still present after fix")
	}
	// Should be valid Go now
	errs := parseErrors(p, got)
	if len(errs) > 0 {
		t.Errorf("still has parse errors after fix: %v", errs)
	}
}

func TestFixTrailingSemicolon(t *testing.T) {
	src := `package main

func foo() {
	x := 1;
}
`
	p := writeTempGo(t, src)
	result, err := fix(p)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("expected success, got remaining errors: %v", result.RemainingErrs)
	}

	got, _ := os.ReadFile(p)
	lines := strings.Split(string(got), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasSuffix(trimmed, ";") && !strings.HasPrefix(trimmed, "//") {
			t.Errorf("trailing semicolon still present: %q", line)
		}
	}
}

func TestFixStatementsOnSameLine(t *testing.T) {
	src := `package main

func foo() {
	if true {
		return
	}    x := 1
}
`
	p := writeTempGo(t, src)
	result, err := fix(p)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("expected success, got remaining errors: %v", result.RemainingErrs)
	}
	if len(result.FixesApplied) == 0 {
		t.Fatal("expected fixes to be applied")
	}

	got, _ := os.ReadFile(p)
	errs := parseErrors(p, got)
	if len(errs) > 0 {
		t.Errorf("still has parse errors: %v", errs)
	}
}

func TestFixSemicolonBeforeAssignment(t *testing.T) {
	src := `package main

func foo() {
	user;; := "test"
}
`
	p := writeTempGo(t, src)
	result, err := fix(p)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("expected success, got remaining errors: %v", result.RemainingErrs)
	}

	got, _ := os.ReadFile(p)
	if strings.Contains(string(got), ";;") {
		t.Error("semicolons before assignment still present")
	}
}

func TestFixCombinedErrors(t *testing.T) {
	// Mimics the kind of garbled output Gemma 4 produces
	src := `package main

import "fmt"

func HandleRoomChat() {
	roomIDVal := Get("current_room_id")
	if roomIDVal == nil {
		return
	}    roomID := roomIDVal.(int);;

	fmt.Println(roomID)
}

func HandleRoomMove() {
	roomIDVal := Get("current_room_id");;
	roomID := roomIDVal.(int);;
	fmt.Println(roomID)
}
`
	p := writeTempGo(t, src)
	result, err := fix(p)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("expected success, got remaining errors: %v", result.RemainingErrs)
	}
	if len(result.FixesApplied) < 3 {
		t.Errorf("expected at least 3 fixes, got %d: %v", len(result.FixesApplied), result.FixesApplied)
	}

	got, _ := os.ReadFile(p)
	if strings.Contains(string(got), ";;") {
		t.Error("double semicolons still present")
	}
	errs := parseErrors(p, got)
	if len(errs) > 0 {
		t.Errorf("still has parse errors: %v", errs)
	}
}

func TestFixAlreadyClean(t *testing.T) {
	src := `package main

func foo() {
	x := 1
}
`
	p := writeTempGo(t, src)
	result, err := fix(p)
	if err != nil {
		t.Fatal(err)
	}
	// No fixes needed, but should succeed (file is already clean)
	if len(result.FixesApplied) != 0 {
		t.Errorf("expected no fixes on clean file, got: %v", result.FixesApplied)
	}
}

func TestFixUnfixableErrors(t *testing.T) {
	// Completely garbled — fix can't help
	src := `package main

func {{{ completely broken
`
	p := writeTempGo(t, src)
	result, err := fix(p)
	if err != nil {
		t.Fatal(err)
	}
	if result.Success {
		t.Error("expected failure on completely broken file")
	}
	if len(result.RemainingErrs) == 0 {
		t.Error("expected remaining errors to be reported")
	}
}

func TestFixPreservesForLoopSemicolons(t *testing.T) {
	src := `package main

func foo() {
	for i := 0; i < 10; i++ {
	}
}
`
	p := writeTempGo(t, src)
	original, _ := os.ReadFile(p)
	result, err := fix(p)
	if err != nil {
		t.Fatal(err)
	}

	got, _ := os.ReadFile(p)
	// For-loop semicolons should be preserved
	if !strings.Contains(string(got), "for i := 0; i < 10; i++") {
		t.Error("for-loop semicolons were incorrectly removed")
	}
	if len(result.FixesApplied) != 0 {
		t.Errorf("expected no fixes on valid for-loop, got: %v", result.FixesApplied)
	}
	// File should be unchanged
	errs := parseErrors(p, original)
	if len(errs) > 0 {
		t.Errorf("original was already broken: %v", errs)
	}
}

func TestFixCorruptedIdentifier(t *testing.T) {
	// Model comma-stuffed a function name: "func handlerFunc,handlerFunc("
	src := `package main

func handlerFunc,handlerFunc(w http.ResponseWriter, r *http.Request) {
}
`
	p := writeTempGo(t, src)
	result, err := fix(p)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("expected success, got remaining errors: %v", result.RemainingErrs)
	}

	got, _ := os.ReadFile(p)
	// The comma-stuffed name should be cleaned up
	if strings.Contains(string(got), "handlerFunc,handlerFunc") {
		t.Error("corrupted function name still present")
	}
	// Should contain just "handlerFunc" once as the function name
	if !strings.Contains(string(got), "func handlerFunc(") {
		t.Errorf("expected 'func handlerFunc(' in output, got:\n%s", string(got))
	}
	errs := parseErrors(p, got)
	if len(errs) > 0 {
		t.Errorf("still has parse errors: %v\nFile:\n%s", errs, string(got))
	}
}

func TestFixCorruptedTypeName(t *testing.T) {
	// Model comma-stuffed a type name: "type Config,Config struct"
	src := `package main

type Config,Config struct {
	Port int
}
`
	p := writeTempGo(t, src)
	result, err := fix(p)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("expected success, got remaining errors: %v", result.RemainingErrs)
	}

	got, _ := os.ReadFile(p)
	if strings.Contains(string(got), "Config,Config") {
		t.Error("corrupted type name still present")
	}
	errs := parseErrors(p, got)
	if len(errs) > 0 {
		t.Errorf("still has parse errors: %v\nFile:\n%s", errs, string(got))
	}
}

func TestFixProducesDiff(t *testing.T) {
	src := `package main

func foo() {
	x := 1;;
}
`
	p := writeTempGo(t, src)
	result, err := fix(p)
	if err != nil {
		t.Fatal(err)
	}
	if result.Diff == "" {
		t.Error("expected diff output when fixes are applied")
	}
}
