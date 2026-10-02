package jsonfile

import (
	"encoding/json"
	"fmt"
	"os"
)

// Load reads the JSON file at path and decodes it into a value of type T.
func Load[T any](path string) (T, error) {
	var value T

	content, err := os.ReadFile(path)
	if err != nil {
		return value, fmt.Errorf("read file: %w", err)
	}

	if err := json.Unmarshal(content, &value); err != nil {
		return value, fmt.Errorf("parse %s: %w", path, err)
	}

	return value, nil
}
