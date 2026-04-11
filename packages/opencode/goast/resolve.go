package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"strconv"
	"strings"
)

// ResolvedTarget holds the resolved AST node and context.
type ResolvedTarget struct {
	Node     ast.Node
	Parent   ast.Node // parent node (e.g., FuncDecl for a stmt target)
	StmtList *[]ast.Stmt // pointer to the statement list (for body targets)
}

var compoundPathRe = regexp.MustCompile(`^(.+)\.(if|for)\[(\d+)\](\.else)?$`)

// parseOccurrence extracts a #N occurrence suffix from a name.
// "Shape" → ("Shape", 1), "Shape#2" → ("Shape", 2), "Shape#abc" → ("Shape#abc", 1)
func parseOccurrence(name string) (string, int) {
	if idx := strings.LastIndex(name, "#"); idx > 0 {
		if n, err := strconv.Atoi(name[idx+1:]); err == nil && n > 0 {
			return name[:idx], n
		}
	}
	return name, 1
}

// ResolveTarget resolves a dotted target path to an AST node.
// Supports #N occurrence suffix for disambiguation when multiple declarations
// share the same name (e.g., "Shape#2" targets the 2nd declaration named "Shape").
// Occurrences are counted in file order across all declaration kinds.
func ResolveTarget(file *ast.File, fset *token.FileSet, target string) (*ResolvedTarget, error) {
	// Check for compound path (e.g., "Func.if[1]", "Func.for[0]", "Func.if[1].else")
	if m := compoundPathRe.FindStringSubmatch(target); m != nil {
		return resolveCompoundTarget(file, fset, m[1], m[2], m[3], m[4] == ".else")
	}

	parts := strings.SplitN(target, ".", 2)
	firstName, occurrence := parseOccurrence(parts[0])

	matchCount := 0
	var lastErr error

	// Single pass through file.Decls in file order, counting occurrences globally.
	// For dotted targets (e.g., "S.Foo"), if a sub-target resolution fails on one
	// match, we continue looking — the sub-target might exist on a different
	// declaration with the same name (e.g., a struct field vs. a method).
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Recv == nil {
				// Package-level function
				if d.Name.Name == firstName {
					matchCount++
					if matchCount == occurrence {
						if len(parts) == 1 {
							return &ResolvedTarget{Node: d}, nil
						}
						result, err := resolveFuncSub(d, parts[1])
						if err == nil {
							return result, nil
						}
						// Sub-target failed (e.g., no such param) — continue looking
						lastErr = err
						matchCount--
					}
				}
			} else {
				// Method: ReceiverType.MethodName
				recvType := baseReceiverType(d.Recv.List[0].Type)
				if recvType == firstName && len(parts) == 2 {
					methodParts := strings.SplitN(parts[1], ".", 2)
					if d.Name.Name == methodParts[0] {
						matchCount++
						if matchCount == occurrence {
							if len(methodParts) == 1 {
								return &ResolvedTarget{Node: d}, nil
							}
							return resolveFuncSub(d, methodParts[1])
						}
					}
				}
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if s.Name.Name == firstName {
						matchCount++
						if matchCount == occurrence {
							if len(parts) == 1 {
								return &ResolvedTarget{Node: s, Parent: d}, nil
							}
							result, err := resolveTypeSub(s, parts[1])
							if err == nil {
								return result, nil
							}
							// Sub-target failed (e.g., no such field) — continue looking
							// for a method or another type that has this sub-target.
							lastErr = err
							matchCount--
						}
					}
				case *ast.ValueSpec:
					for _, name := range s.Names {
						if name.Name == firstName && len(parts) == 1 {
							matchCount++
							if matchCount == occurrence {
								return &ResolvedTarget{Node: s, Parent: d}, nil
							}
						}
					}
				}
			}
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}
	if matchCount > 0 {
		return nil, fmt.Errorf("target %q: only %d declaration(s) named %q found (requested #%d)", target, matchCount, firstName, occurrence)
	}
	return nil, fmt.Errorf("target %q not found", target)
}

func resolveFuncSub(decl *ast.FuncDecl, sub string) (*ResolvedTarget, error) {
	// Check parameters
	if decl.Type.Params != nil {
		for _, p := range decl.Type.Params.List {
			for _, n := range p.Names {
				if n.Name == sub {
					return &ResolvedTarget{Node: p, Parent: decl}, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("sub-target %q not found in function %s — sub-targets must be parameter names. For duplicate declarations with the same name, use %s#N to target the Nth occurrence (e.g., %s#2)", sub, decl.Name.Name, decl.Name.Name, decl.Name.Name)
}

func resolveTypeSub(spec *ast.TypeSpec, sub string) (*ResolvedTarget, error) {
	switch t := spec.Type.(type) {
	case *ast.StructType:
		if t.Fields != nil {
			for _, f := range t.Fields.List {
				for _, n := range f.Names {
					if n.Name == sub {
						return &ResolvedTarget{Node: f, Parent: spec}, nil
					}
				}
			}
		}
	case *ast.InterfaceType:
		if t.Methods != nil {
			for _, m := range t.Methods.List {
				for _, n := range m.Names {
					if n.Name == sub {
						return &ResolvedTarget{Node: m, Parent: spec}, nil
					}
				}
			}
		}
	}
	// Provide guidance: sub-targets are field/method names, not line numbers.
	// For duplicate declarations, use Name#N syntax (e.g., Shape#2).
	return nil, fmt.Errorf("sub-target %q not found in type %s — sub-targets must be field or method names. For duplicate declarations with the same name, use %s#N to target the Nth occurrence (e.g., %s#2)", sub, spec.Name.Name, spec.Name.Name, spec.Name.Name)
}

func resolveCompoundTarget(file *ast.File, fset *token.FileSet, funcTarget, stmtKind, indexStr string, isElse bool) (*ResolvedTarget, error) {
	idx, err := strconv.Atoi(indexStr)
	if err != nil {
		return nil, fmt.Errorf("invalid index %q", indexStr)
	}

	// Resolve the function first
	funcResolved, err := ResolveTarget(file, fset, funcTarget)
	if err != nil {
		return nil, err
	}

	var funcDecl *ast.FuncDecl
	switch n := funcResolved.Node.(type) {
	case *ast.FuncDecl:
		funcDecl = n
	default:
		return nil, fmt.Errorf("target %q is not a function", funcTarget)
	}

	if funcDecl.Body == nil || idx >= len(funcDecl.Body.List) {
		return nil, fmt.Errorf("statement index %d out of range in %s (has %d statements)", idx, funcTarget, len(funcDecl.Body.List))
	}

	stmt := funcDecl.Body.List[idx]

	switch stmtKind {
	case "if":
		ifStmt, ok := stmt.(*ast.IfStmt)
		if !ok {
			return nil, fmt.Errorf("statement at index %d in %s is not an if statement", idx, funcTarget)
		}
		if isElse {
			if ifStmt.Else == nil {
				return nil, fmt.Errorf("if statement at index %d in %s has no else block", idx, funcTarget)
			}
			elseBlock, ok := ifStmt.Else.(*ast.BlockStmt)
			if !ok {
				return nil, fmt.Errorf("else block at index %d in %s is not a block statement", idx, funcTarget)
			}
			return &ResolvedTarget{Node: elseBlock, Parent: ifStmt, StmtList: &elseBlock.List}, nil
		}
		return &ResolvedTarget{Node: ifStmt.Body, Parent: ifStmt, StmtList: &ifStmt.Body.List}, nil
	case "for":
		rangeStmt, ok := stmt.(*ast.RangeStmt)
		if !ok {
			// Also handle regular for statements
			forStmt, ok := stmt.(*ast.ForStmt)
			if !ok {
				return nil, fmt.Errorf("statement at index %d in %s is not a for statement", idx, funcTarget)
			}
			return &ResolvedTarget{Node: forStmt.Body, Parent: forStmt, StmtList: &forStmt.Body.List}, nil
		}
		return &ResolvedTarget{Node: rangeStmt.Body, Parent: rangeStmt, StmtList: &rangeStmt.Body.List}, nil
	}

	return nil, fmt.Errorf("unknown compound statement kind: %s", stmtKind)
}
