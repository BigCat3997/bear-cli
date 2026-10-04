package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

type LoadOptions struct {
	// Path normally comes from the --config flag. Leave empty to use
	// BEAR_CONFIG or the platform-specific default path.
	Path string

	// Overrides normally contains values supplied through Cobra flags.
	Overrides Overrides

	// LookupEnv can be replaced during testing. Leave nil to use os.LookupEnv.
	LookupEnv EnvLookup
}

type Result struct {
	Config Config
	Path   string

	// FileFound reports whether a configuration file was loaded. A missing
	// default configuration file is not considered an error.
	FileFound bool
}

// Load reads, resolves, normalizes, and validates Bear configuration.
func Load(options LoadOptions) (Result, error) {
	lookupEnv := options.LookupEnv
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}

	configPath, err := ResolvePath(options.Path, lookupEnv)
	if err != nil {
		return Result{}, err
	}

	cfg := Default()

	fileFound, err := loadFile(configPath, &cfg)
	if err != nil {
		return Result{}, err
	}

	// When --config or BEAR_CONFIG explicitly selects a file, a missing file is
	// likely a user mistake and should be reported.
	if !fileFound && pathWasExplicit(options.Path, lookupEnv) {
		return Result{}, fmt.Errorf(
			"configuration file %q does not exist",
			configPath,
		)
	}

	cfg = Resolve(cfg, options.Overrides, lookupEnv)

	if err := cfg.Validate(); err != nil {
		return Result{}, fmt.Errorf(
			"validate configuration %q: %w",
			configPath,
			err,
		)
	}

	return Result{
		Config:    cfg,
		Path:      configPath,
		FileFound: fileFound,
	}, nil
}

func loadFile(path string, cfg *Config) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}

		return false, fmt.Errorf(
			"read configuration file %q: %w",
			path,
			err,
		)
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return false, fmt.Errorf(
			"decode configuration file %q: %w",
			path,
			err,
		)
	}

	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]Profile)
	}

	return true, nil
}

func pathWasExplicit(path string, lookupEnv EnvLookup) bool {
	if path != "" {
		return true
	}

	value, found := lookupEnv("BEAR_CONFIG")
	return found && value != ""
}
