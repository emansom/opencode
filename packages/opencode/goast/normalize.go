package main

import (
	"encoding/json"
	"strings"
	"unicode"
)

// KnownOps is the set of valid operation names. Each maps to an individual
// tool in the FC format (e.g. "create_function" → "go_create_function").
var KnownOps = map[string]bool{
	"create_function": true, "create_method": true, "create_struct": true,
	"create_interface": true, "create_type_alias": true, "create_named_type": true,
	"add_const": true, "add_var": true, "delete": true,
	"add_parameter": true, "remove_parameter": true, "set_return_types": true,
	"change_receiver": true, "set_function_name": true,
	"clear_body": true, "remove_statement": true,
	"insert_call": true, "insert_method_call": true,
	"insert_assign_call": true, "insert_assign_value": true, "insert_var_decl": true,
	"insert_return": true, "insert_if": true, "insert_if_init": true,
	"insert_for_range": true, "insert_defer": true, "insert_go": true,
	"insert_error_check": true, "push_arg": true, "set_else": true,
	"add_struct_field": true, "remove_struct_field": true, "set_field_type": true,
	"set_field_name": true, "set_struct_tag": true, "remove_struct_tag": true,
	"add_interface_method": true, "remove_interface_method": true, "add_interface_embed": true,
	"add_import": true, "remove_import": true,
	"set_const_value": true, "set_var_value": true, "set_const_type": true, "set_var_type": true,
	"remove_var": true, "remove_const": true,
	"set_package": true, "set_build_constraint": true, "add_generate_directive": true,
	"set_doc_comment": true, "remove_doc_comment": true, "add_line_comment": true,
	"rename": true, "extract_interface": true,
	"gopls_rename": true, "organize_imports": true,
}

// SanitizeIdentifier cleans a Go identifier that may have been corrupted by
// the model. Handles comma-stuffed values ("handlerFunc,handlerFunc" → "handlerFunc")
// and strips characters invalid in Go identifiers.
func SanitizeIdentifier(s string) string {
	if s == "" {
		return s
	}
	// Take first comma-separated value — model sometimes doubles or comma-stuffs.
	if idx := strings.IndexByte(s, ','); idx > 0 {
		s = s[:idx]
	}
	s = strings.TrimSpace(s)
	// Strip characters that aren't valid in Go identifiers.
	// Track whether we've started the identifier (seen a letter/underscore)
	// so we skip leading digits.
	var buf strings.Builder
	started := false
	for _, r := range s {
		if r == '_' || unicode.IsLetter(r) {
			buf.WriteRune(r)
			started = true
		} else if started && unicode.IsDigit(r) {
			buf.WriteRune(r)
		}
	}
	result := buf.String()
	if result == "" {
		return s // Return original if sanitization removed everything
	}
	return result
}

// sanitizeCommaStuffedField takes a field value that may contain embedded
// key:value pairs from the model cramming multiple params into one field.
// Returns the cleaned value and any extracted extra key:value pairs.
// Example: "handlerFunc,func:fmt.Println" → "handlerFunc", [("func","fmt.Println")]
func sanitizeCommaStuffedField(val string) (string, []fieldPair) {
	if !strings.Contains(val, ",") {
		return val, nil
	}
	parts := strings.Split(val, ",")
	clean := parts[0]
	var extras []fieldPair
	for _, part := range parts[1:] {
		k, v, ok := strings.Cut(part, ":")
		if ok && k != "" {
			extras = append(extras, fieldPair{k: strings.TrimSpace(k), v: strings.TrimSpace(v)})
		}
		// If no colon, it might be a duplicated value — ignore it
	}
	return clean, extras
}

type fieldPair struct{ k, v string }

// ParseToolCallJSON converts raw JSON arguments from a model's tool call into
// an Operation struct. It handles model confusion patterns such as using the op
// name as a JSON key instead of as the value of the "op" field.
//
// Example confusion: {"add_struct_field":"Config","fieldName":"Debug",...}
// instead of: {"op":"add_struct_field","target":"Config","fieldName":"Debug",...}
func ParseToolCallJSON(raw map[string]interface{}) Operation {
	// Map "filePath" → "file" for the Go struct's json tag.
	if fp, ok := raw["filePath"].(string); ok {
		raw["file"] = fp
		delete(raw, "filePath")
	}

	// Pre-clean: detect comma-stuffed field values and extract embedded params.
	// Example: {"file":"handlerFunc,func:fmt.Println"} → file=handlerFunc + func=fmt.Println
	for key, val := range raw {
		s, ok := val.(string)
		if !ok || key == "args" || key == "params" || key == "returns" || key == "valueArgs" || key == "tag" || key == "text" || key == "directive" || key == "constraint" {
			continue // These fields legitimately contain commas
		}
		cleaned, extras := sanitizeCommaStuffedField(s)
		if len(extras) > 0 {
			raw[key] = cleaned
			for _, kv := range extras {
				if _, exists := raw[kv.k]; !exists {
					raw[kv.k] = kv.v
				}
			}
		}
	}

	// Use json round-trip: the Operation struct has json tags matching
	// every field name, so this handles all standard fields in one step.
	data, _ := json.Marshal(raw)
	var op Operation
	json.Unmarshal(data, &op)
	op.Mode = "edit"

	// Sanitize identifier fields — prevent file corruption from comma-stuffed
	// or invalid-character names like "handlerFunc,handlerFunc".
	op.Name = SanitizeIdentifier(op.Name)
	op.FieldName = SanitizeIdentifier(op.FieldName)
	op.ReceiverVar = SanitizeIdentifier(op.ReceiverVar)
	op.Method = SanitizeIdentifier(op.Method)
	op.MethodName = SanitizeIdentifier(op.MethodName)

	// Fix filePath/target confusion: if file doesn't look like a .go file path,
	// the model probably put a function/type name there instead.
	if op.File != "" && !strings.HasSuffix(op.File, ".go") && !strings.Contains(op.File, "/") && !strings.Contains(op.File, "\\") {
		if op.Target == "" {
			op.Target = op.File
		}
		op.File = "" // Will be caught by the caller with a helpful error
	}

	// Detect comma-stuffed op: "op":"insert_assign_call,target:X,vars:Y"
	// The model sometimes crams multiple params into the op field as CSV.
	// Extract the real op name (before first comma) and parse extra pairs.
	if op.Op != "" && strings.Contains(op.Op, ",") {
		parts := strings.Split(op.Op, ",")
		if KnownOps[parts[0]] {
			op.Op = parts[0]
			for _, kv := range parts[1:] {
				k, v, ok := strings.Cut(kv, ":")
				if !ok {
					continue
				}
				applyExtraField(&op, k, v)
			}
		}
	}

	// Detect op-as-key pattern: {"add_struct_field":"Config",...}
	// The model puts the operation name as a JSON key with the target as value.
	if op.Op == "" {
		for key, val := range raw {
			if KnownOps[key] {
				op.Op = key
				if s, ok := val.(string); ok && op.Target == "" {
					op.Target = s
				}
				break
			}
		}
	}

	// Last resort: infer op from the fields that are set.
	InferOp(&op)

	return op
}

// applyExtraField sets an Operation field by key name. Used when the model
// stuffs extra key:value pairs into the op field as CSV.
func applyExtraField(op *Operation, key, val string) {
	switch key {
	case "target":
		if op.Target == "" {
			op.Target = val
		}
	case "name":
		if op.Name == "" {
			op.Name = val
		}
	case "vars":
		if op.Vars == "" {
			op.Vars = val
		}
	case "receiver":
		if op.Receiver == "" {
			op.Receiver = val
		}
	case "method":
		if op.Method == "" {
			op.Method = val
		}
	case "func":
		if op.Func == "" {
			op.Func = val
		}
	case "short":
		if op.Short == nil && val == "true" {
			b := true
			op.Short = &b
		}
	case "fieldName":
		if op.FieldName == "" {
			op.FieldName = val
		}
	case "fieldType":
		if op.FieldType == "" {
			op.FieldType = val
		}
	case "receiverType":
		if op.ReceiverType == "" {
			op.ReceiverType = val
		}
	case "receiverVar":
		if op.ReceiverVar == "" {
			op.ReceiverVar = val
		}
	case "params":
		if op.Params == "" {
			op.Params = val
		}
	case "returns":
		if op.Returns == "" {
			op.Returns = val
		}
	case "args":
		if op.Args == "" {
			op.Args = val
		}
	}
}

// InferOp attempts to guess the operation from the fields that are set,
// for cases where the model omits the "op" field entirely.
func InferOp(op *Operation) {
	if op.Op != "" {
		return
	}
	// create_method: has receiverType + name
	if op.ReceiverType != "" && op.Name != "" {
		op.Op = "create_method"
		return
	}
	// create_function: has name + (params or returns) but no receiverType
	if op.Name != "" && (op.Params != "" || op.Returns != "") && op.ReceiverType == "" {
		op.Op = "create_function"
		return
	}
}

// NormalizeOp resolves commonly confused parameter names.
// Gemma 4 (especially smaller variants) sometimes uses:
//   - "method" instead of "name" for create_method
//   - "receiver" instead of "receiverType" for create_method
//   - "target" instead of "name" for create_struct/create_interface/create_function
//   - "Type.Field" as target for add_struct_field (should be just "Type")
func NormalizeOp(op *Operation) {
	switch op.Op {
	case "create_method":
		// "method" → "name" (the method being created)
		if op.Name == "" && op.Method != "" {
			op.Name = op.Method
			op.Method = ""
		}
		// "receiver" → "receiverType" (the type, not the variable)
		if op.ReceiverType == "" && op.Receiver != "" {
			op.ReceiverType = op.Receiver
			op.Receiver = ""
		}
	case "create_function":
		// "method" → "name"
		if op.Name == "" && op.Method != "" {
			op.Name = op.Method
			op.Method = ""
		}
		// "target" → "name" (some models put the function name in target)
		if op.Name == "" && op.Target != "" {
			op.Name = op.Target
			op.Target = ""
		}
	case "create_struct", "create_interface":
		// "target" → "name" (the struct/interface being created)
		if op.Name == "" && op.Target != "" {
			op.Name = op.Target
			op.Target = ""
		}
	case "add_struct_field":
		// "Struct.NewField" target → strip the field part to just "Struct"
		if op.Target != "" && op.FieldName != "" && strings.Contains(op.Target, ".") {
			parts := strings.SplitN(op.Target, ".", 2)
			if parts[1] == op.FieldName {
				op.Target = parts[0]
			}
		}
	}
}
