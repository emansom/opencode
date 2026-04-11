package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GoplsResult is the output of a gopls-powered operation.
type GoplsResult struct {
	Success bool     `json:"success"`
	Diff    string   `json:"diff,omitempty"`
	Content string   `json:"content,omitempty"`
	Errors  []string `json:"errors,omitempty"`
	// For rename: list of files modified
	ModifiedFiles []string `json:"modifiedFiles,omitempty"`
}

// goplsAvailable checks if gopls is installed and accessible.
func goplsAvailable() bool {
	_, err := exec.LookPath("gopls")
	return err == nil
}

// goplsRename performs a cross-package rename using gopls.
// This is the correct way to rename exported symbols that may be
// referenced across multiple packages in the module.
func goplsRename(op Operation) (*GoplsResult, error) {
	if op.Target == "" {
		return nil, fmt.Errorf("target is required for rename")
	}
	if op.NewName == "" {
		return nil, fmt.Errorf("newName is required for rename")
	}

	if !goplsAvailable() {
		return nil, fmt.Errorf("gopls not found — install with: go install golang.org/x/tools/gopls@latest")
	}

	// Read the file to find the byte offset of the target symbol
	src, err := os.ReadFile(op.File)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	original := string(src)

	offset, err := findSymbolOffset(op.File, src, op.Target)
	if err != nil {
		return nil, fmt.Errorf("find symbol %q: %w", op.Target, err)
	}

	// gopls rename uses file:line:col format
	// Convert byte offset to line:col
	line, col := offsetToLineCol(src, offset)
	position := fmt.Sprintf("%s:%d:%d", op.File, line, col)

	// Run gopls rename — it modifies files in-place
	cmd := exec.Command("gopls", "rename", "-w", position, op.NewName)
	cmd.Dir = findModuleRoot(op.File)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("gopls rename: %s\n%s", err, string(output))
	}

	// Read back the modified file
	modified, err := os.ReadFile(op.File)
	if err != nil {
		return nil, fmt.Errorf("read modified file: %w", err)
	}

	// Collect list of modified files from gopls output
	var modifiedFiles []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasSuffix(line, ".go") {
			modifiedFiles = append(modifiedFiles, line)
		}
	}

	diff := computeDiff(original, string(modified))

	return &GoplsResult{
		Success:       true,
		Diff:          diff,
		Content:       string(modified),
		ModifiedFiles: modifiedFiles,
	}, nil
}

// goplsOrganizeImports runs gopls to organize imports (add missing, remove unused).
func goplsOrganizeImports(op Operation) (*GoplsResult, error) {
	if !goplsAvailable() {
		return nil, fmt.Errorf("gopls not found — install with: go install golang.org/x/tools/gopls@latest")
	}

	src, err := os.ReadFile(op.File)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	original := string(src)

	// Use gopls codeaction to organize imports
	cmd := exec.Command("gopls", "codeaction", "-exec", "-kind", "source.organizeImports", op.File)
	cmd.Dir = findModuleRoot(op.File)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback: try goimports if gopls codeaction fails
		return goimportsFallback(op.File, original)
	}

	_ = output // gopls applies the edit

	modified, err := os.ReadFile(op.File)
	if err != nil {
		return nil, fmt.Errorf("read modified file: %w", err)
	}

	diff := computeDiff(original, string(modified))
	if diff == "" {
		return &GoplsResult{
			Success: true,
			Content: original,
		}, nil
	}

	return &GoplsResult{
		Success: true,
		Diff:    diff,
		Content: string(modified),
	}, nil
}

// goimportsFallback uses goimports as a simpler alternative.
func goimportsFallback(file string, original string) (*GoplsResult, error) {
	goimports, err := exec.LookPath("goimports")
	if err != nil {
		// Last resort: just run gofmt
		return nil, fmt.Errorf("neither gopls codeaction nor goimports available")
	}

	cmd := exec.Command(goimports, "-w", file)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("goimports: %s\n%s", err, string(output))
	}

	modified, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	diff := computeDiff(original, string(modified))
	return &GoplsResult{
		Success: true,
		Diff:    diff,
		Content: string(modified),
	}, nil
}

// goplsReferences finds all references to a symbol across the module.
func goplsReferences(file string, target string) ([]string, error) {
	if !goplsAvailable() {
		return nil, fmt.Errorf("gopls not available")
	}

	src, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	offset, err := findSymbolOffset(file, src, target)
	if err != nil {
		return nil, err
	}

	line, col := offsetToLineCol(src, offset)
	position := fmt.Sprintf("%s:%d:%d", file, line, col)

	cmd := exec.Command("gopls", "references", position)
	cmd.Dir = findModuleRoot(file)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("gopls references: %s", string(output))
	}

	var refs []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			refs = append(refs, line)
		}
	}
	return refs, nil
}

// goplsImplementations finds implementations of an interface.
func goplsImplementations(file string, target string) ([]string, error) {
	if !goplsAvailable() {
		return nil, fmt.Errorf("gopls not available")
	}

	src, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	offset, err := findSymbolOffset(file, src, target)
	if err != nil {
		return nil, err
	}

	line, col := offsetToLineCol(src, offset)
	position := fmt.Sprintf("%s:%d:%d", file, line, col)

	cmd := exec.Command("gopls", "implementation", position)
	cmd.Dir = findModuleRoot(file)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("gopls implementation: %s", string(output))
	}

	var impls []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			impls = append(impls, line)
		}
	}
	return impls, nil
}

// GoplsSignatureResult contains the signature help for a function/method.
type GoplsSignatureResult struct {
	Signature string   `json:"signature"`
	Doc       string   `json:"doc,omitempty"`
	Params    []string `json:"params,omitempty"`
}

// --- Helpers ---

// findSymbolOffset finds the byte offset of a named symbol in Go source.
// Supports dotted targets like "Server.Start", "Config.Port".
func findSymbolOffset(file string, src []byte, target string) (int, error) {
	// Parse the target path
	parts := strings.SplitN(target, ".", 2)
	topName := parts[0]

	// Strip occurrence suffix (e.g., "Name#2")
	if idx := strings.LastIndex(topName, "#"); idx >= 0 {
		topName = topName[:idx]
	}

	content := string(src)

	if len(parts) == 1 {
		// Top-level name — find "func topName", "type topName", "var topName", "const topName"
		patterns := []string{
			"func " + topName + "(",
			"func " + topName + " ",
			"type " + topName + " ",
			topName + " =",
			topName + " ",
		}
		for _, pat := range patterns {
			idx := strings.Index(content, pat)
			if idx >= 0 {
				return idx + strings.Index(pat, topName), nil
			}
		}
		return -1, fmt.Errorf("symbol %q not found in %s", target, file)
	}

	// Dotted path: find the sub-name after locating the parent
	subName := parts[1]
	parentOffset, err := findSymbolOffset(file, src, topName)
	if err != nil {
		return -1, err
	}

	// Search for the sub-name after the parent
	rest := content[parentOffset:]
	idx := strings.Index(rest, subName)
	if idx < 0 {
		return -1, fmt.Errorf("sub-symbol %q not found after %q", subName, topName)
	}
	return parentOffset + idx, nil
}

// offsetToLineCol converts a byte offset to 1-based line and column.
func offsetToLineCol(src []byte, offset int) (line, col int) {
	line = 1
	col = 1
	for i := 0; i < offset && i < len(src); i++ {
		if src[i] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return
}

// findModuleRoot walks up from file to find the directory containing go.mod.
func findModuleRoot(file string) string {
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return filepath.Dir(file) // fallback
		}
		dir = parent
	}
}

// GoplsCodeActionResult wraps the output from a gopls code action.
type GoplsCodeActionResult struct {
	Title string          `json:"title"`
	Kind  string          `json:"kind"`
	Edit  json.RawMessage `json:"edit,omitempty"`
}
