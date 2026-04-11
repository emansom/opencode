// prompt.go — generates system prompt and tool definitions from the operation registry.
//
// This is the single source of truth for tool metadata. Both OpenCode (TypeScript)
// and integration tests call this at runtime instead of hardcoding tool descriptions.
//
// Gemma 4 was trained on Google's FC (Function Calling) format with flat tool
// calling structures: one tool per action, each with only its own human-readable
// parameters. The registry encodes this structure — each entry maps to one tool
// with a distinct name and a minimal, self-contained parameter set.

package main

import (
	"fmt"
	"strings"
)

// ToolParam describes a single parameter of a tool.
type ToolParam struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"` // "string", "number", "boolean"
	Description string   `json:"description"`
	Required    bool     `json:"required"`
	Enum        []string `json:"enum,omitempty"`
}

// ToolInfo describes a single tool generated from the operation registry.
type ToolInfo struct {
	Name        string      `json:"name"`        // e.g. "go_create_function"
	Op          string      `json:"op"`           // e.g. "create_function"
	Description string      `json:"description"`
	Params      []ToolParam `json:"params"`
}

// PromptResult is the output of the "prompt" mode.
type PromptResult struct {
	SystemPrompt string     `json:"systemPrompt"`
	Tools        []ToolInfo `json:"tools"`
}

// opRegistry defines all operations, their tool names, descriptions, and params.
// This is the single source of truth — OpenCode and tests derive everything from here.
//
// Each entry corresponds to one tool in Gemma 4's FC format. Parameters listed
// here are ONLY the ones relevant to that specific operation. The filePath param
// is always added automatically and is not listed here.
var opRegistry = []ToolInfo{
	// --- Declaration creation ---
	{
		Name: "go_create_function", Op: "create_function",
		Description: "Create a new Go function. Body starts empty — add statements with go_insert_call, go_insert_return, etc.",
		Params: []ToolParam{
			{Name: "name", Type: "string", Description: "Function name", Required: true},
			{Name: "params", Type: "string", Description: "Parameters as name:type pairs, comma-separated. Example: ctx:context.Context,id:string"},
			{Name: "returns", Type: "string", Description: "Return types, comma-separated. Example: *Config,error"},
			{Name: "position", Type: "string", Description: "Where to place: first, last, before, after, at_index", Enum: []string{"first", "last", "before", "after", "at_index"}},
			{Name: "anchor", Type: "string", Description: "Reference declaration for before/after positioning"},
		},
	},
	{
		Name: "go_create_method", Op: "create_method",
		Description: "Create a new method on a Go type. Body starts empty.",
		Params: []ToolParam{
			{Name: "name", Type: "string", Description: "Method name", Required: true},
			{Name: "receiverType", Type: "string", Description: "Go type of receiver, e.g. *Server or Server", Required: true},
			{Name: "receiverVar", Type: "string", Description: "Receiver variable name, e.g. s. Derived from type if omitted"},
			{Name: "params", Type: "string", Description: "Parameters as name:type pairs, comma-separated"},
			{Name: "returns", Type: "string", Description: "Return types, comma-separated"},
			{Name: "position", Type: "string", Description: "Where to place: first, last, before, after, at_index", Enum: []string{"first", "last", "before", "after", "at_index"}},
			{Name: "anchor", Type: "string", Description: "Reference declaration for before/after positioning"},
		},
	},
	{
		Name: "go_create_struct", Op: "create_struct",
		Description: "Create a new empty struct type.",
		Params: []ToolParam{
			{Name: "name", Type: "string", Description: "Struct name", Required: true},
			{Name: "position", Type: "string", Description: "Where to place: first, last, before, after, at_index", Enum: []string{"first", "last", "before", "after", "at_index"}},
			{Name: "anchor", Type: "string", Description: "Reference declaration for before/after positioning"},
		},
	},
	{
		Name: "go_create_interface", Op: "create_interface",
		Description: "Create a new empty interface type.",
		Params: []ToolParam{
			{Name: "name", Type: "string", Description: "Interface name", Required: true},
			{Name: "position", Type: "string", Description: "Where to place: first, last, before, after, at_index", Enum: []string{"first", "last", "before", "after", "at_index"}},
			{Name: "anchor", Type: "string", Description: "Reference declaration for before/after positioning"},
		},
	},
	{
		Name: "go_create_type_alias", Op: "create_type_alias",
		Description: "Create a type alias: type Name = TargetType.",
		Params: []ToolParam{
			{Name: "name", Type: "string", Description: "Alias name", Required: true},
			{Name: "targetType", Type: "string", Description: "Target type, e.g. http.Handler", Required: true},
		},
	},
	{
		Name: "go_create_named_type", Op: "create_named_type",
		Description: "Create a named type: type Name UnderlyingType.",
		Params: []ToolParam{
			{Name: "name", Type: "string", Description: "Type name", Required: true},
			{Name: "underlyingType", Type: "string", Description: "Underlying type, e.g. int, string, map[string]interface{}", Required: true},
		},
	},
	{
		Name: "go_add_const", Op: "add_const",
		Description: "Add a const declaration.",
		Params: []ToolParam{
			{Name: "name", Type: "string", Description: "Const name", Required: true},
			{Name: "type", Type: "string", Description: "Go type (optional for typed consts)"},
			{Name: "valueSpec", Type: "string", Description: "Flat value: int:42, string:hello, ident:x"},
			{Name: "valueFunc", Type: "string", Description: "Function name for call-valued const: errors.New"},
			{Name: "valueArgs", Type: "string", Description: "Comma-separated flat args for valueFunc: string:not found"},
			{Name: "block", Type: "string", Description: "Const block name to add to"},
		},
	},
	{
		Name: "go_add_var", Op: "add_var",
		Description: "Add a var declaration.",
		Params: []ToolParam{
			{Name: "name", Type: "string", Description: "Variable name", Required: true},
			{Name: "type", Type: "string", Description: "Go type"},
			{Name: "valueSpec", Type: "string", Description: "Flat value: int:42, nil, ident:x"},
			{Name: "valueFunc", Type: "string", Description: "Function name for call-valued var"},
			{Name: "valueArgs", Type: "string", Description: "Comma-separated flat args for valueFunc"},
			{Name: "block", Type: "string", Description: "Var block name to add to"},
		},
	},
	{
		Name: "go_delete", Op: "delete",
		Description: "Delete a top-level declaration (function, method, type, const, var).",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Dotted path of declaration to delete", Required: true},
		},
	},

	// --- Function/method signature ---
	{
		Name: "go_add_parameter", Op: "add_parameter",
		Description: "Add a parameter to a function or method signature.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Function or method path, e.g. Start or Server.Start", Required: true},
			{Name: "name", Type: "string", Description: "Parameter name", Required: true},
			{Name: "type", Type: "string", Description: "Parameter type", Required: true},
			{Name: "position", Type: "string", Description: "Where to place: first, last, before, after", Enum: []string{"first", "last", "before", "after"}},
			{Name: "anchor", Type: "string", Description: "Reference parameter name for before/after positioning"},
		},
	},
	{
		Name: "go_remove_parameter", Op: "remove_parameter",
		Description: "Remove a parameter from a function or method signature.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Parameter path, e.g. Start.ctx or Server.Start.id", Required: true},
		},
	},
	{
		Name: "go_set_return_types", Op: "set_return_types",
		Description: "Set the return types of a function or method.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Function or method path", Required: true},
			{Name: "returns", Type: "string", Description: "Comma-separated return types: *Config,error", Required: true},
		},
	},
	{
		Name: "go_change_receiver", Op: "change_receiver",
		Description: "Change the receiver type of a method.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Method path, e.g. Server.Start", Required: true},
			{Name: "receiverType", Type: "string", Description: "New receiver type, e.g. *Server", Required: true},
		},
	},
	{
		Name: "go_set_function_name", Op: "set_function_name",
		Description: "Rename a function or method (local file only, use go_gopls_rename for cross-package).",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Current function or method path", Required: true},
			{Name: "name", Type: "string", Description: "New name", Required: true},
		},
	},

	// --- Function/method body ---
	{
		Name: "go_clear_body", Op: "clear_body",
		Description: "Remove all statements from a function or method body.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Function or method path", Required: true},
		},
	},
	{
		Name: "go_remove_statement", Op: "remove_statement",
		Description: "Remove a statement from a function body by index.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Function or method path", Required: true},
			{Name: "index", Type: "number", Description: "0-based statement index", Required: true},
		},
	},
	{
		Name: "go_insert_call", Op: "insert_call",
		Description: "Insert a function call statement: func(args...).",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Function or method path, or block path like Func.if[0]", Required: true},
			{Name: "func", Type: "string", Description: "Function to call, e.g. fmt.Println, os.ReadFile", Required: true},
			{Name: "args", Type: "string", Description: "Comma-separated flat args: string:hello,ident:name"},
			{Name: "position", Type: "string", Description: "Where to insert: first, last, at_index", Enum: []string{"first", "last", "at_index"}},
			{Name: "index", Type: "number", Description: "Statement index for at_index positioning"},
		},
	},
	{
		Name: "go_insert_method_call", Op: "insert_method_call",
		Description: "Insert a method call statement: receiver.method(args...).",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Function or method path, or block path", Required: true},
			{Name: "receiver", Type: "string", Description: "Receiver variable, e.g. s, s.listener", Required: true},
			{Name: "method", Type: "string", Description: "Method name, e.g. Close, Start", Required: true},
			{Name: "args", Type: "string", Description: "Comma-separated flat args"},
			{Name: "position", Type: "string", Description: "Where to insert: first, last, at_index", Enum: []string{"first", "last", "at_index"}},
			{Name: "index", Type: "number", Description: "Statement index for at_index positioning"},
		},
	},
	{
		Name: "go_insert_assign_call", Op: "insert_assign_call",
		Description: "Insert an assignment from a function or method call: vars := func(args...) or vars := receiver.method(args...).",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Function or method path, or block path", Required: true},
			{Name: "vars", Type: "string", Description: "Comma-separated LHS variable names: ln,err", Required: true},
			{Name: "func", Type: "string", Description: "Function to call (use this OR receiver+method)"},
			{Name: "receiver", Type: "string", Description: "Receiver variable for method call"},
			{Name: "method", Type: "string", Description: "Method name for method call"},
			{Name: "args", Type: "string", Description: "Comma-separated flat args"},
			{Name: "short", Type: "boolean", Description: "true for := short declaration, false for ="},
			{Name: "position", Type: "string", Description: "Where to insert: first, last, at_index", Enum: []string{"first", "last", "at_index"}},
			{Name: "index", Type: "number", Description: "Statement index for at_index positioning"},
		},
	},
	{
		Name: "go_insert_assign_value", Op: "insert_assign_value",
		Description: "Insert an assignment from a value: varName := valueSpec.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Function or method path, or block path", Required: true},
			{Name: "varName", Type: "string", Description: "Variable name", Required: true},
			{Name: "valueSpec", Type: "string", Description: "Flat value: int:42, nil, ident:x, string:hello", Required: true},
			{Name: "position", Type: "string", Description: "Where to insert: first, last, at_index", Enum: []string{"first", "last", "at_index"}},
			{Name: "index", Type: "number", Description: "Statement index for at_index positioning"},
		},
	},
	{
		Name: "go_insert_var_decl", Op: "insert_var_decl",
		Description: "Insert a var declaration inside a function body: var varName varType.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Function or method path, or block path", Required: true},
			{Name: "varName", Type: "string", Description: "Variable name", Required: true},
			{Name: "varType", Type: "string", Description: "Go type", Required: true},
			{Name: "position", Type: "string", Description: "Where to insert: first, last, at_index", Enum: []string{"first", "last", "at_index"}},
			{Name: "index", Type: "number", Description: "Statement index for at_index positioning"},
		},
	},
	{
		Name: "go_insert_return", Op: "insert_return",
		Description: "Insert a return statement.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Function or method path, or block path", Required: true},
			{Name: "values", Type: "string", Description: "Comma-separated flat return values: nil,ident:err"},
			{Name: "position", Type: "string", Description: "Where to insert: first, last, at_index", Enum: []string{"first", "last", "at_index"}},
			{Name: "index", Type: "number", Description: "Statement index for at_index positioning"},
		},
	},
	{
		Name: "go_insert_if", Op: "insert_if",
		Description: "Insert an if statement with a comparison condition.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Function or method path, or block path", Required: true},
			{Name: "condLeft", Type: "string", Description: "Left side of comparison: ident:err, nil, int:0", Required: true},
			{Name: "operator", Type: "string", Description: "Comparison operator", Required: true, Enum: []string{"==", "!=", "<", ">", "<=", ">="}},
			{Name: "condRight", Type: "string", Description: "Right side of comparison: nil, ident:expected", Required: true},
			{Name: "position", Type: "string", Description: "Where to insert: first, last, at_index", Enum: []string{"first", "last", "at_index"}},
			{Name: "index", Type: "number", Description: "Statement index for at_index positioning"},
		},
	},
	{
		Name: "go_insert_if_init", Op: "insert_if_init",
		Description: "Insert an if statement with init: if vars := func(args); left op right { }.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Function or method path, or block path", Required: true},
			{Name: "initVars", Type: "string", Description: "Comma-separated init assignment variables: err", Required: true},
			{Name: "initFunc", Type: "string", Description: "Function to call in init", Required: true},
			{Name: "condLeft", Type: "string", Description: "Left side of comparison", Required: true},
			{Name: "operator", Type: "string", Description: "Comparison operator", Required: true, Enum: []string{"==", "!=", "<", ">", "<=", ">="}},
			{Name: "condRight", Type: "string", Description: "Right side of comparison", Required: true},
			{Name: "initArgs", Type: "string", Description: "Comma-separated flat args for init call"},
			{Name: "position", Type: "string", Description: "Where to insert: first, last, at_index", Enum: []string{"first", "last", "at_index"}},
			{Name: "index", Type: "number", Description: "Statement index for at_index positioning"},
		},
	},
	{
		Name: "go_insert_for_range", Op: "insert_for_range",
		Description: "Insert a for-range loop: for key, value := range iterable { }.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Function or method path, or block path", Required: true},
			{Name: "iterable", Type: "string", Description: "Flat iterable: ident:items, selector:s.clients", Required: true},
			{Name: "key", Type: "string", Description: "Key variable name (use _ to discard)"},
			{Name: "value", Type: "string", Description: "Value variable name (use _ to discard)"},
			{Name: "position", Type: "string", Description: "Where to insert: first, last, at_index", Enum: []string{"first", "last", "at_index"}},
			{Name: "index", Type: "number", Description: "Statement index for at_index positioning"},
		},
	},
	{
		Name: "go_insert_defer", Op: "insert_defer",
		Description: "Insert a defer statement: defer func(args...).",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Function or method path, or block path", Required: true},
			{Name: "func", Type: "string", Description: "Function to defer, e.g. conn.Close", Required: true},
			{Name: "args", Type: "string", Description: "Comma-separated flat args"},
			{Name: "position", Type: "string", Description: "Where to insert: first, last, at_index", Enum: []string{"first", "last", "at_index"}},
			{Name: "index", Type: "number", Description: "Statement index for at_index positioning"},
		},
	},
	{
		Name: "go_insert_go", Op: "insert_go",
		Description: "Insert a go statement: go func(args...).",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Function or method path, or block path", Required: true},
			{Name: "func", Type: "string", Description: "Function to call in goroutine", Required: true},
			{Name: "args", Type: "string", Description: "Comma-separated flat args"},
			{Name: "position", Type: "string", Description: "Where to insert: first, last, at_index", Enum: []string{"first", "last", "at_index"}},
			{Name: "index", Type: "number", Description: "Statement index for at_index positioning"},
		},
	},
	{
		Name: "go_insert_error_check", Op: "insert_error_check",
		Description: "Insert an error check: if errVar != nil { return returnValues }.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Function or method path, or block path", Required: true},
			{Name: "errVar", Type: "string", Description: "Error variable name, e.g. err", Required: true},
			{Name: "returnValues", Type: "string", Description: "Comma-separated flat return values: nil,ident:err", Required: true},
			{Name: "position", Type: "string", Description: "Where to insert: first, last, at_index", Enum: []string{"first", "last", "at_index"}},
			{Name: "index", Type: "number", Description: "Statement index for at_index positioning"},
		},
	},
	{
		Name: "go_push_arg", Op: "push_arg",
		Description: "Append an argument to an existing function call.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Function or method path", Required: true},
			{Name: "stmtIndex", Type: "number", Description: "Statement index containing the call", Required: true},
			{Name: "arg", Type: "string", Description: "Flat argument: selector:s.port, string:hello, ident:x", Required: true},
			{Name: "callIndex", Type: "number", Description: "Call index within return/assign (default 0)"},
		},
	},
	{
		Name: "go_set_else", Op: "set_else",
		Description: "Add an else block to an existing if statement.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Function or method path", Required: true},
			{Name: "stmtIndex", Type: "number", Description: "Statement index of the if statement", Required: true},
		},
	},

	// --- Struct field editing ---
	{
		Name: "go_add_struct_field", Op: "add_struct_field",
		Description: "Add a field to a struct.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Struct name (e.g. Config, NOT Config.FieldName)", Required: true},
			{Name: "fieldName", Type: "string", Description: "Field name", Required: true},
			{Name: "fieldType", Type: "string", Description: "Go type, e.g. int, string, *http.Handler", Required: true},
			{Name: "tag", Type: "string", Description: "Full struct tag string, e.g. `json:\"name\"`"},
			{Name: "position", Type: "string", Description: "Where to place: first, last, before, after", Enum: []string{"first", "last", "before", "after"}},
			{Name: "anchor", Type: "string", Description: "Reference field name for before/after positioning"},
		},
	},
	{
		Name: "go_remove_struct_field", Op: "remove_struct_field",
		Description: "Remove a field from a struct.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Field path, e.g. Config.Port", Required: true},
		},
	},
	{
		Name: "go_set_field_type", Op: "set_field_type",
		Description: "Change the type of a struct field.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Field path, e.g. Config.Port", Required: true},
			{Name: "fieldType", Type: "string", Description: "New Go type", Required: true},
		},
	},
	{
		Name: "go_set_field_name", Op: "set_field_name",
		Description: "Rename a struct field (local file only).",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Field path, e.g. Config.Port", Required: true},
			{Name: "fieldName", Type: "string", Description: "New field name", Required: true},
		},
	},
	{
		Name: "go_set_struct_tag", Op: "set_struct_tag",
		Description: "Set a struct tag on a field.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Field path, e.g. Config.Port", Required: true},
			{Name: "tagKey", Type: "string", Description: "Tag key: json, db, yaml, etc.", Required: true},
			{Name: "tagValue", Type: "string", Description: "Tag value", Required: true},
		},
	},
	{
		Name: "go_remove_struct_tag", Op: "remove_struct_tag",
		Description: "Remove a struct tag from a field.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Field path, e.g. Config.Port", Required: true},
			{Name: "tagKey", Type: "string", Description: "Tag key to remove"},
		},
	},

	// --- Interface editing ---
	{
		Name: "go_add_interface_method", Op: "add_interface_method",
		Description: "Add a method to an interface.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Interface name", Required: true},
			{Name: "methodName", Type: "string", Description: "Method name", Required: true},
			{Name: "params", Type: "string", Description: "Parameters as name:type pairs, comma-separated"},
			{Name: "returns", Type: "string", Description: "Return types, comma-separated"},
		},
	},
	{
		Name: "go_remove_interface_method", Op: "remove_interface_method",
		Description: "Remove a method from an interface.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Method path, e.g. Reader.Read", Required: true},
		},
	},
	{
		Name: "go_add_interface_embed", Op: "add_interface_embed",
		Description: "Embed a type in an interface.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Interface name", Required: true},
			{Name: "embedType", Type: "string", Description: "Type to embed, e.g. io.Reader", Required: true},
		},
	},

	// --- Import management ---
	{
		Name: "go_add_import", Op: "add_import",
		Description: "Add an import to the file.",
		Params: []ToolParam{
			{Name: "path", Type: "string", Description: "Import path, e.g. fmt, net/http", Required: true},
			{Name: "alias", Type: "string", Description: "Import alias (optional)"},
		},
	},
	{
		Name: "go_remove_import", Op: "remove_import",
		Description: "Remove an import from the file.",
		Params: []ToolParam{
			{Name: "path", Type: "string", Description: "Import path to remove", Required: true},
		},
	},

	// --- Const/var editing ---
	{
		Name: "go_set_const_value", Op: "set_const_value",
		Description: "Change the value of a const.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Const name", Required: true},
			{Name: "valueSpec", Type: "string", Description: "Flat value: int:42, string:hello"},
			{Name: "valueFunc", Type: "string", Description: "Function name for call-valued const"},
			{Name: "valueArgs", Type: "string", Description: "Comma-separated flat args for valueFunc"},
		},
	},
	{
		Name: "go_set_var_value", Op: "set_var_value",
		Description: "Change the value of a var.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Variable name", Required: true},
			{Name: "valueSpec", Type: "string", Description: "Flat value: int:42, nil, ident:x"},
			{Name: "valueFunc", Type: "string", Description: "Function name for call-valued var"},
			{Name: "valueArgs", Type: "string", Description: "Comma-separated flat args for valueFunc"},
		},
	},
	{
		Name: "go_set_const_type", Op: "set_const_type",
		Description: "Change the type of a const.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Const name", Required: true},
			{Name: "type", Type: "string", Description: "New Go type", Required: true},
		},
	},
	{
		Name: "go_set_var_type", Op: "set_var_type",
		Description: "Change the type of a var.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Variable name", Required: true},
			{Name: "type", Type: "string", Description: "New Go type", Required: true},
		},
	},
	{
		Name: "go_remove_var", Op: "remove_var",
		Description: "Remove a var declaration.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Variable name", Required: true},
		},
	},
	{
		Name: "go_remove_const", Op: "remove_const",
		Description: "Remove a const declaration.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Const name", Required: true},
		},
	},

	// --- Package/file level ---
	{
		Name: "go_set_package", Op: "set_package",
		Description: "Change the package name.",
		Params: []ToolParam{
			{Name: "name", Type: "string", Description: "New package name", Required: true},
		},
	},
	{
		Name: "go_set_build_constraint", Op: "set_build_constraint",
		Description: "Set a build constraint (//go:build expression).",
		Params: []ToolParam{
			{Name: "constraint", Type: "string", Description: "Build constraint expression, e.g. linux && amd64", Required: true},
		},
	},
	{
		Name: "go_add_generate_directive", Op: "add_generate_directive",
		Description: "Add a //go:generate directive.",
		Params: []ToolParam{
			{Name: "directive", Type: "string", Description: "Directive content (without //go:generate prefix)", Required: true},
			{Name: "anchor", Type: "string", Description: "Reference declaration for positioning"},
			{Name: "position", Type: "string", Description: "Where to place: before, after", Enum: []string{"before", "after"}},
		},
	},

	// --- Comments ---
	{
		Name: "go_set_doc_comment", Op: "set_doc_comment",
		Description: "Set the doc comment on a declaration.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Declaration path", Required: true},
			{Name: "text", Type: "string", Description: "Comment text (// prefix added automatically)", Required: true},
		},
	},
	{
		Name: "go_remove_doc_comment", Op: "remove_doc_comment",
		Description: "Remove the doc comment from a declaration.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Declaration path", Required: true},
		},
	},
	{
		Name: "go_add_line_comment", Op: "add_line_comment",
		Description: "Add an inline comment to a statement.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Declaration path", Required: true},
			{Name: "text", Type: "string", Description: "Comment text", Required: true},
		},
	},

	// --- Refactoring ---
	{
		Name: "go_rename", Op: "rename",
		Description: "Rename a symbol in the current file only. Use go_gopls_rename for cross-package renames.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Symbol path to rename", Required: true},
			{Name: "newName", Type: "string", Description: "New name", Required: true},
		},
	},
	{
		Name: "go_extract_interface", Op: "extract_interface",
		Description: "Extract an interface from a struct's methods.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Struct name", Required: true},
			{Name: "interfaceName", Type: "string", Description: "Name for the new interface", Required: true},
		},
	},

	// --- Gopls-powered operations ---
	{
		Name: "go_gopls_rename", Op: "gopls_rename",
		Description: "Cross-package rename using gopls. Renames the symbol across all files in the module.",
		Params: []ToolParam{
			{Name: "target", Type: "string", Description: "Symbol path to rename", Required: true},
			{Name: "newName", Type: "string", Description: "New name", Required: true},
		},
	},
	{
		Name: "go_organize_imports", Op: "organize_imports",
		Description: "Auto-add missing imports and remove unused ones using gopls/goimports.",
		Params: []ToolParam{},
	},
}

// generatePrompt builds the system prompt and tool definitions from the operation registry.
func generatePrompt() *PromptResult {
	var sb strings.Builder

	sb.WriteString(`You have Go AST editing tools. Each tool performs one AST operation on a Go source file.
Always call go_inspect first to understand the file structure before editing.

IMPORTANT: Every tool call requires filePath — the file system path to the .go file (e.g. main.go, internal/server/server.go).
The target parameter is DIFFERENT — it identifies WHICH declaration inside the file to edit (e.g. main, Server.Start, Config.Port).

Value encoding uses kind:content format:
- Identifiers: ident:err, ident:ctx, ident:x
- Nil: nil
- Numbers: int:42, float:3.14
- Strings: string:hello
- Selectors: selector:s.Port, selector:cfg.Debug
- Addresses: addr:cfg (produces &cfg)
- Calls: call:fmt.Errorf (inline call in value position)

Target uses dotted paths: FuncName, Type.Method, Struct.Field, Func.if[N], Func.for[N], Func.if[N].else
When multiple declarations share the same name, append #N: Shape#2 targets the 2nd Shape.

Example: Create a function and add a statement to it (two separate tool calls):
  1. go_create_function(filePath="main.go", name="handleRequest", params="w:http.ResponseWriter,r:*http.Request")
  2. go_insert_call(filePath="main.go", target="handleRequest", func="fmt.Fprintf", args="ident:w,string:OK")

`)

	// Group tools by category for the system prompt
	categories := []struct {
		name string
		ops  []string
	}{
		{"Declaration", []string{"create_function", "create_method", "create_struct", "create_interface", "create_type_alias", "create_named_type", "add_const", "add_var", "delete"}},
		{"Signature", []string{"add_parameter", "remove_parameter", "set_return_types", "change_receiver", "set_function_name"}},
		{"Body", []string{"clear_body", "remove_statement", "insert_call", "insert_method_call", "insert_assign_call", "insert_assign_value", "insert_var_decl", "insert_return", "insert_if", "insert_if_init", "insert_for_range", "insert_defer", "insert_go", "insert_error_check", "push_arg", "set_else"}},
		{"Struct", []string{"add_struct_field", "remove_struct_field", "set_field_type", "set_field_name", "set_struct_tag", "remove_struct_tag"}},
		{"Interface", []string{"add_interface_method", "remove_interface_method", "add_interface_embed"}},
		{"Import", []string{"add_import", "remove_import"}},
		{"Const/Var", []string{"set_const_value", "set_var_value", "set_const_type", "set_var_type", "remove_var", "remove_const"}},
		{"File", []string{"set_package", "set_build_constraint", "add_generate_directive"}},
		{"Comment", []string{"set_doc_comment", "remove_doc_comment", "add_line_comment"}},
		{"Refactor", []string{"rename", "extract_interface", "gopls_rename", "organize_imports"}},
	}

	// Build a lookup from op → ToolInfo
	opLookup := make(map[string]*ToolInfo, len(opRegistry))
	for i := range opRegistry {
		opLookup[opRegistry[i].Op] = &opRegistry[i]
	}

	sb.WriteString("Available tools:\n")
	for _, cat := range categories {
		sb.WriteString(fmt.Sprintf("\n%s:\n", cat.name))
		for _, opName := range cat.ops {
			info := opLookup[opName]
			if info == nil {
				continue
			}
			// Tool name and required params
			var reqParams []string
			var optParams []string
			for _, p := range info.Params {
				if p.Required {
					reqParams = append(reqParams, p.Name)
				} else {
					optParams = append(optParams, p.Name)
				}
			}
			line := fmt.Sprintf("  %s(filePath", info.Name)
			for _, r := range reqParams {
				line += ", " + r
			}
			if len(optParams) > 0 {
				line += " [, " + strings.Join(optParams, ", ") + "]"
			}
			line += fmt.Sprintf(") — %s", info.Description)
			sb.WriteString(line + "\n")
		}
	}

	return &PromptResult{
		SystemPrompt: sb.String(),
		Tools:        opRegistry,
	}
}
