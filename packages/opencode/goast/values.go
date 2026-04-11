package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

// ParseFlatValue parses a flat "kind:content" string into an AST expression.
func ParseFlatValue(s string) (ast.Expr, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty value")
	}

	// Handle special cases without colon
	switch s {
	case "nil":
		return ast.NewIdent("nil"), nil
	case "true":
		return ast.NewIdent("true"), nil
	case "false":
		return ast.NewIdent("false"), nil
	}

	// Split on first colon
	idx := strings.IndexByte(s, ':')
	if idx < 0 {
		return nil, fmt.Errorf("invalid flat value (no kind): %q", s)
	}

	kind := s[:idx]
	content := s[idx+1:]

	switch kind {
	case "ident":
		if content == "" {
			return nil, fmt.Errorf("ident kind requires a name")
		}
		return ast.NewIdent(content), nil

	case "int":
		return &ast.BasicLit{Kind: token.INT, Value: content}, nil

	case "float":
		return &ast.BasicLit{Kind: token.FLOAT, Value: content}, nil

	case "string":
		return &ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("%q", content)}, nil

	case "selector":
		parts := strings.SplitN(content, ".", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("selector requires dotted path: %q", content)
		}
		return &ast.SelectorExpr{
			X:   ast.NewIdent(parts[0]),
			Sel: ast.NewIdent(parts[1]),
		}, nil

	case "addr":
		return &ast.UnaryExpr{
			Op: token.AND,
			X:  ast.NewIdent(content),
		}, nil

	case "call":
		// Zero-arg function call. Arguments added via push_arg.
		fn := parseFuncExpr(content)
		return &ast.CallExpr{Fun: fn}, nil

	default:
		return nil, fmt.Errorf("unknown value kind: %q", kind)
	}
}

// ParseFlatValues parses a comma-separated string of flat values.
func ParseFlatValues(s string) ([]ast.Expr, error) {
	if s == "" {
		return nil, nil
	}
	parts := splitCSV(s)
	var exprs []ast.Expr
	for _, p := range parts {
		expr, err := ParseFlatValue(p)
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, expr)
	}
	return exprs, nil
}

// ParseFlatParams parses a comma-separated "name:type" string into AST fields.
func ParseFlatParams(s string) ([]*ast.Field, error) {
	if s == "" {
		return nil, nil
	}
	parts := splitCSV(s)
	var fields []*ast.Field
	for _, p := range parts {
		p = strings.TrimSpace(p)
		idx := strings.IndexByte(p, ':')
		if idx < 0 {
			return nil, fmt.Errorf("parameter must be name:type, got %q", p)
		}
		name := p[:idx]
		typ := p[idx+1:]
		fields = append(fields, &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(name)},
			Type:  parseTypeExpr(typ),
		})
	}
	return fields, nil
}

// ParseReturnTypes parses a comma-separated list of return types.
func ParseReturnTypes(s string) (*ast.FieldList, error) {
	if s == "" {
		return nil, nil
	}
	parts := splitCSV(s)
	var fields []*ast.Field
	for _, p := range parts {
		p = strings.TrimSpace(p)
		fields = append(fields, &ast.Field{
			Type: parseTypeExpr(p),
		})
	}
	return &ast.FieldList{List: fields}, nil
}

// parseFuncExpr parses a function name (possibly dotted) into an AST expression.
func parseFuncExpr(name string) ast.Expr {
	parts := strings.SplitN(name, ".", 2)
	if len(parts) == 2 {
		return &ast.SelectorExpr{
			X:   ast.NewIdent(parts[0]),
			Sel: ast.NewIdent(parts[1]),
		}
	}
	return ast.NewIdent(name)
}

// parseTypeExpr parses a Go type string into an AST expression.
func parseTypeExpr(s string) ast.Expr {
	s = strings.TrimSpace(s)

	if strings.HasPrefix(s, "*") {
		return &ast.StarExpr{X: parseTypeExpr(s[1:])}
	}
	if strings.HasPrefix(s, "[]") {
		return &ast.ArrayType{Elt: parseTypeExpr(s[2:])}
	}
	if strings.HasPrefix(s, "map[") {
		// Simple map parsing: map[K]V
		depth := 1
		i := 4
		for i < len(s) && depth > 0 {
			if s[i] == '[' {
				depth++
			} else if s[i] == ']' {
				depth--
			}
			i++
		}
		key := s[4 : i-1]
		val := s[i:]
		return &ast.MapType{
			Key:   parseTypeExpr(key),
			Value: parseTypeExpr(val),
		}
	}
	if strings.HasPrefix(s, "...") {
		return &ast.Ellipsis{Elt: parseTypeExpr(s[3:])}
	}
	if s == "interface{}" {
		return &ast.InterfaceType{Methods: &ast.FieldList{}}
	}

	// Dotted name (e.g. "context.Context")
	if idx := strings.IndexByte(s, '.'); idx > 0 {
		return &ast.SelectorExpr{
			X:   ast.NewIdent(s[:idx]),
			Sel: ast.NewIdent(s[idx+1:]),
		}
	}

	return ast.NewIdent(s)
}

// splitCSV splits a string by commas, but respects parentheses and brackets.
func splitCSV(s string) []string {
	var parts []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	parts = append(parts, strings.TrimSpace(s[start:]))
	return parts
}
