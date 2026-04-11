package main

import (
	"encoding/json"
	"strings"
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

	// Use json round-trip: the Operation struct has json tags matching
	// every field name, so this handles all standard fields in one step.
	data, _ := json.Marshal(raw)
	var op Operation
	json.Unmarshal(data, &op)
	op.Mode = "edit"

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
