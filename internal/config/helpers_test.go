package config

import (
	"os"
	"path/filepath"
	"testing"
)

// envFrom builds an EnvLookup backed by a map. Keys present in the map are
// "found" even when their value is empty, mirroring os.LookupEnv.
func envFrom(values map[string]string) EnvLookup {
	return func(key string) (string, bool) {
		value, found := values[key]
		return value, found
	}
}

func strPtr(s string) *string { return &s }

func writeConfigFile(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
