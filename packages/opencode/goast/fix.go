package main

import (
	"bytes"
	"fmt"
	"go/format"
	"go/parser"
	"go/scanner"
	"go/token"
	"os"
	"regexp"
	"strings"
)

// FixResult is the output of a fix operation.
type FixResult struct {
	Success       bool     `json:"success"`
	Diff          string   `json:"diff,omitempty"`
	Content       string   `json:"content,omitempty"`
	FixesApplied  []string `json:"fixesApplied,omitempty"`
	RemainingErrs []string `json:"remainingErrors,omitempty"`
}

// fix attempts to automatically repair syntax errors in a Go source file.
// It analyzes parse errors and applies targeted fixes iteratively until the
// file parses cleanly or no more automatic fixes can be applied.
func fix(filename string) (*FixResult, error) {
	src, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read file: %v", err)
	}

	original := string(src)
	current := original
	var allFixes []string

	// Phase 1: Pre-parse cleanup — fix obviously wrong patterns that may or
	// may not cause parse errors but are never valid Go style.
	cleaned, preFixes := preParseCleanup(current)
	if len(preFixes) > 0 {
		allFixes = append(allFixes, preFixes...)
		current = cleaned
	}

	// Phase 2: Error-driven fixes — parse, analyze errors, apply targeted
	// fixes, repeat until clean or stuck.
	for round := 0; round < 20; round++ {
		errs := parseErrors(filename, []byte(current))
		if len(errs) == 0 {
			break // Clean parse — done
		}

		fixed, fixes := applyFixes(current, errs)
		if len(fixes) == 0 {
			break // No more auto-fixable errors
		}
		allFixes = append(allFixes, fixes...)
		current = fixed
	}

	// Phase 3: gofmt — canonicalize formatting and remove any remaining
	// unnecessary semicolons that the parser tolerates.
	formatted, fmtErr := format.Source([]byte(current))
	if fmtErr == nil {
		current = string(formatted)
	}

	// Check for remaining errors
	remaining := parseErrors(filename, []byte(current))
	var remainingStrs []string
	for _, e := range remaining {
		remainingStrs = append(remainingStrs, e.Error())
	}

	if current == original && len(allFixes) == 0 {
		return &FixResult{
			Success:       false,
			RemainingErrs: remainingStrs,
		}, nil
	}

	diff := computeDiff(original, current)

	// Write the fixed file
	if err := os.WriteFile(filename, []byte(current), 0644); err != nil {
		return nil, fmt.Errorf("write file: %v", err)
	}

	return &FixResult{
		Success:       len(remaining) == 0,
		Diff:          diff,
		Content:       current,
		FixesApplied:  allFixes,
		RemainingErrs: remainingStrs,
	}, nil
}

// parseErrors returns the list of syntax errors from parsing src.
func parseErrors(filename string, src []byte) []scanner.Error {
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, filename, src, parser.AllErrors|parser.ParseComments)
	if err == nil {
		return nil
	}
	if list, ok := err.(scanner.ErrorList); ok {
		var out []scanner.Error
		for _, e := range list {
			out = append(out, *e)
		}
		return out
	}
	// Single error
	return []scanner.Error{{Msg: err.Error()}}
}

// applyFixes analyzes parse errors and applies all safe automatic fixes.
// Returns the fixed source and a list of human-readable fix descriptions.
func applyFixes(src string, errs []scanner.Error) (string, []string) {
	lines := strings.Split(src, "\n")
	var fixes []string

	// Track which lines we've already modified this round to avoid conflicts
	modified := map[int]bool{}

	for _, e := range errs {
		lineIdx := e.Pos.Line - 1 // 0-based
		if lineIdx < 0 || lineIdx >= len(lines) {
			continue
		}
		if modified[lineIdx] {
			continue
		}

		line := lines[lineIdx]
		col := e.Pos.Column - 1 // 0-based

		switch {
		case fixDoubleSemicolon(lines, lineIdx, line, &fixes):
			modified[lineIdx] = true

		case fixTrailingSemicolon(lines, lineIdx, line, &fixes):
			modified[lineIdx] = true

		case fixStatementsOnSameLine(lines, lineIdx, line, col, e.Msg, &fixes):
			modified[lineIdx] = true

		case fixMissingSemicolon(lines, lineIdx, line, col, e.Msg, &fixes):
			modified[lineIdx] = true

		case fixExtraSemicolonBeforeIdent(lines, lineIdx, line, col, e.Msg, &fixes):
			modified[lineIdx] = true

		case fixUnexpectedSemicolon(lines, lineIdx, line, col, e.Msg, &fixes):
			modified[lineIdx] = true

		case fixExpectedComma(lines, lineIdx, line, col, e.Msg, &fixes):
			modified[lineIdx] = true
		}
	}

	result := strings.Join(lines, "\n")
	if result == src {
		return src, nil
	}
	return result, fixes
}

// preParseCleanup applies pattern-based fixes for obviously wrong constructs
// that may or may not cause parse errors. These are never valid Go style:
// double semicolons, semicolons before assignments, etc.
func preParseCleanup(src string) (string, []string) {
	lines := strings.Split(src, "\n")
	var fixes []string

	for i, line := range lines {
		// Double semicolons: ";;" → "" (Go never needs explicit semicolons)
		if strings.Contains(line, ";;") {
			lines[i] = strings.ReplaceAll(line, ";;", "")
			fixes = append(fixes, fmt.Sprintf("line %d: removed double semicolons", i+1))
			continue
		}

		// Semicolons jammed before := or = assignment: "user; :=" → "user :="
		reSemiAssign := regexp.MustCompile(`(\w+)\s*;+\s*(:?=)`)
		if reSemiAssign.MatchString(line) {
			lines[i] = reSemiAssign.ReplaceAllString(line, "$1 $2")
			fixes = append(fixes, fmt.Sprintf("line %d: removed semicolons before assignment", i+1))
			continue
		}
	}

	result := strings.Join(lines, "\n")
	return result, fixes
}

// --- Individual fix functions ---
// Each returns true if it applied a fix.

// fixDoubleSemicolon removes duplicate semicolons: ";;" → ""
// Go uses implicit semicolons, so explicit ";;" is always wrong.
func fixDoubleSemicolon(lines []string, idx int, line string, fixes *[]string) bool {
	if !strings.Contains(line, ";;") {
		return false
	}
	lines[idx] = strings.ReplaceAll(line, ";;", "")
	*fixes = append(*fixes, fmt.Sprintf("line %d: removed double semicolons", idx+1))
	return true
}

// fixTrailingSemicolon removes a single trailing semicolon at end of line.
// Go doesn't use explicit semicolons; they're inserted by the lexer.
func fixTrailingSemicolon(lines []string, idx int, line string, fixes *[]string) bool {
	trimmed := strings.TrimRight(line, " \t")
	if !strings.HasSuffix(trimmed, ";") {
		return false
	}
	// Don't remove semicolons inside for-loop headers or multi-statement lines that look intentional
	stripped := strings.TrimSpace(line)
	if strings.HasPrefix(stripped, "for ") || strings.HasPrefix(stripped, "for(") {
		return false
	}
	// Don't touch single-line if/else with semicolons (init statements)
	if strings.HasPrefix(stripped, "if ") && strings.Count(stripped, ";") == 1 {
		return false
	}
	lines[idx] = strings.TrimRight(trimmed, ";")
	*fixes = append(*fixes, fmt.Sprintf("line %d: removed trailing semicolon", idx+1))
	return true
}

// fixStatementsOnSameLine splits "} stmt" into two lines.
// e.g. "}    roomID := roomIDVal.(int)" → "}\n\troomID := roomIDVal.(int)"
func fixStatementsOnSameLine(lines []string, idx int, line string, col int, msg string, fixes *[]string) bool {
	// Look for pattern: closing brace followed by a statement on the same line
	re := regexp.MustCompile(`^(\s*\})\s+(\S.*)$`)
	m := re.FindStringSubmatch(line)
	if m == nil {
		return false
	}
	closeBrace := m[1]
	rest := m[2]
	// Don't split if rest is "else" — that's valid Go
	restTrimmed := strings.TrimSpace(rest)
	if strings.HasPrefix(restTrimmed, "else") {
		return false
	}
	// Determine indentation from the closing brace line + one tab
	indent := extractIndent(closeBrace) + "\t"
	lines[idx] = closeBrace + "\n" + indent + rest
	*fixes = append(*fixes, fmt.Sprintf("line %d: split statement from closing brace onto new line", idx+1))
	return true
}

// fixMissingSemicolon handles "expected ';', found X" — often means a newline
// is needed between two statements that got merged.
func fixMissingSemicolon(lines []string, idx int, line string, col int, msg string, fixes *[]string) bool {
	if !strings.Contains(msg, "expected ';'") {
		return false
	}
	// If the error column points to a position in the middle of a line where
	// two statements are jammed together, insert a newline
	if col <= 0 || col >= len(line) {
		return false
	}
	// Check if there's a valid Go token boundary at/near the column
	before := strings.TrimRight(line[:col], " \t")
	after := strings.TrimLeft(line[col:], " \t")
	if before == "" || after == "" {
		return false
	}
	// Only split if 'before' ends with something that could end a statement
	// (identifier, literal, ), ], etc.) and 'after' starts with something
	// that could start a statement (identifier, keyword, etc.)
	lastChar := before[len(before)-1]
	if lastChar != ')' && lastChar != ']' && lastChar != '"' && lastChar != '`' &&
		!isIdentChar(lastChar) && !isDigit(lastChar) {
		return false
	}
	indent := extractIndent(line)
	lines[idx] = before + "\n" + indent + after
	*fixes = append(*fixes, fmt.Sprintf("line %d: split merged statements at column %d", idx+1, col+1))
	return true
}

// fixExtraSemicolonBeforeIdent handles patterns like "user;; :=" where a
// semicolon appears before a := or = assignment.
func fixExtraSemicolonBeforeIdent(lines []string, idx int, line string, col int, msg string, fixes *[]string) bool {
	// Match patterns like "varname;; :=" or "varname; :="
	re := regexp.MustCompile(`(\w+)\s*;+\s*(:?=)`)
	if !re.MatchString(line) {
		return false
	}
	lines[idx] = re.ReplaceAllString(line, "$1 $2")
	*fixes = append(*fixes, fmt.Sprintf("line %d: removed semicolons before assignment operator", idx+1))
	return true
}

// fixUnexpectedSemicolon handles "expected X, found ';'" errors by removing
// the offending semicolon.
func fixUnexpectedSemicolon(lines []string, idx int, line string, col int, msg string, fixes *[]string) bool {
	if !strings.Contains(msg, "found ';'") {
		return false
	}
	if col <= 0 || col > len(line) {
		return false
	}
	// Remove the semicolon at the specified column
	if col-1 < len(line) && line[col-1] == ';' {
		lines[idx] = line[:col-1] + line[col:]
		*fixes = append(*fixes, fmt.Sprintf("line %d: removed unexpected semicolon at column %d", idx+1, col))
		return true
	}
	return false
}

// fixExpectedComma handles "expected ',', found X" in struct literals,
// function args, etc. by inserting a comma.
func fixExpectedComma(lines []string, idx int, line string, col int, msg string, fixes *[]string) bool {
	if !strings.Contains(msg, "expected ','") {
		return false
	}
	if col <= 0 || col > len(line) {
		return false
	}
	// Insert comma before the column position
	insertAt := col - 1
	if insertAt > len(line) {
		insertAt = len(line)
	}
	lines[idx] = line[:insertAt] + "," + line[insertAt:]
	*fixes = append(*fixes, fmt.Sprintf("line %d: inserted missing comma at column %d", idx+1, col))
	return true
}

// --- Helpers ---

func extractIndent(line string) string {
	var buf bytes.Buffer
	for _, ch := range line {
		if ch == ' ' || ch == '\t' {
			buf.WriteRune(ch)
		} else {
			break
		}
	}
	return buf.String()
}

func isIdentChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_' || (b >= '0' && b <= '9')
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}
