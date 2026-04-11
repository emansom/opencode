package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	var op Operation
	if err := json.NewDecoder(os.Stdin).Decode(&op); err != nil {
		writeError(fmt.Sprintf("failed to decode input: %v", err))
		return
	}

	// "prompt" mode does not require a file — it generates tool definitions
	// and system prompt from the operation registry.
	if op.Mode == "prompt" {
		result := generatePrompt()
		json.NewEncoder(os.Stdout).Encode(result)
		return
	}

	if op.File == "" {
		writeError("file path is required")
		return
	}

	switch op.Mode {
	case "inspect":
		result, err := inspect(op.File)
		if err != nil {
			writeError(fmt.Sprintf("inspect failed: %v", err))
			return
		}
		json.NewEncoder(os.Stdout).Encode(result)

	case "edit":
		if op.Op == "" {
			writeError("op is required for edit mode")
			return
		}
		result, err := edit(op)
		if err != nil {
			writeError(fmt.Sprintf("edit failed: %v", err))
			return
		}
		json.NewEncoder(os.Stdout).Encode(result)

	case "fix":
		result, err := fix(op.File)
		if err != nil {
			writeError(fmt.Sprintf("fix failed: %v", err))
			return
		}
		json.NewEncoder(os.Stdout).Encode(result)

	default:
		writeError(fmt.Sprintf("unknown mode: %q", op.Mode))
	}
}

func writeError(msg string) {
	result := EditResult{
		Success: false,
		Errors:  []string{msg},
	}
	json.NewEncoder(os.Stdout).Encode(result)
}
