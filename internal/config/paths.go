package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	ApplicationDirectory = "bear"
	ConfigFilename       = "config.toml"
)

// ConfigDirectory returns Bear's configuration directory at ~/.config/bear.
// Keeping one location across the CLI and desktop app makes the active
// configuration predictable on macOS as well as other platforms.
func ConfigDirectory() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determine home directory: %w", err)
	}

	return filepath.Join(homeDir, ".config", ApplicationDirectory), nil
}

// DefaultPath returns config.toml in the current working directory, which is
// the project root when Bear is run from there.
func DefaultPath() (string, error) {
	path, err := filepath.Abs(ConfigFilename)
	if err != nil {
		return "", fmt.Errorf("resolve default configuration path: %w", err)
	}

	return path, nil
}

// ResolvePath applies the following path precedence:
//
//  1. --config or another explicit path
//  2. BEAR_CONFIG
//  3. config.toml in the current working directory
func ResolvePath(explicitPath string, lookupEnv EnvLookup) (string, error) {
	if explicitPath != "" {
		return expandPath(explicitPath)
	}

	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}

	if value, found := lookupEnv("BEAR_CONFIG"); found && value != "" {
		return expandPath(value)
	}

	return DefaultPath()
}

// expandPath supports paths beginning with ~/.
func expandPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("configuration path must not be empty")
	}

	if path == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("determine home directory: %w", err)
		}

		return homeDir, nil
	}

	homePrefix := "~" + string(filepath.Separator)
	if len(path) >= len(homePrefix) && path[:len(homePrefix)] == homePrefix {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("determine home directory: %w", err)
		}

		return filepath.Join(homeDir, path[len(homePrefix):]), nil
	}

	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve configuration path %q: %w", path, err)
	}

	return absolutePath, nil
}
