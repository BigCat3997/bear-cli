package config

import (
	"path/filepath"
	"testing"
)

func TestDefaultPathIsConfigFileInWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath() error = %v", err)
	}
	// Resolve symlinks (e.g. /var -> /private/var on macOS) before comparing.
	got, _ := filepath.EvalSymlinks(filepath.Dir(path))
	want, _ := filepath.EvalSymlinks(dir)
	if got != want || filepath.Base(path) != "config.toml" {
		t.Fatalf("DefaultPath() = %q, want config.toml in %q", path, want)
	}
}

func TestResolvePathPrecedence(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	lookup := func(key string) (string, bool) {
		if key == "BEAR_CONFIG" {
			return "~/from-environment.toml", true
		}
		return "", false
	}

	path, err := ResolvePath("~/explicit.toml", lookup)
	if err != nil {
		t.Fatalf("ResolvePath(explicit) error = %v", err)
	}
	if want := filepath.Join(home, "explicit.toml"); path != want {
		t.Fatalf("explicit path = %q, want %q", path, want)
	}

	path, err = ResolvePath("", lookup)
	if err != nil {
		t.Fatalf("ResolvePath(environment) error = %v", err)
	}
	if want := filepath.Join(home, "from-environment.toml"); path != want {
		t.Fatalf("environment path = %q, want %q", path, want)
	}
}

func TestConfigDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir, err := ConfigDirectory()
	if err != nil {
		t.Fatalf("ConfigDirectory() error = %v", err)
	}
	if want := filepath.Join(home, ".config", "bear"); dir != want {
		t.Fatalf("ConfigDirectory() = %q, want %q", dir, want)
	}
}

func TestResolvePathFallsBackToDefault(t *testing.T) {
	t.Chdir(t.TempDir())

	for name, lookup := range map[string]EnvLookup{
		"unset":     func(string) (string, bool) { return "", false },
		"empty env": func(string) (string, bool) { return "", true },
	} {
		t.Run(name, func(t *testing.T) {
			got, err := ResolvePath("", lookup)
			if err != nil {
				t.Fatalf("ResolvePath error = %v", err)
			}
			want, _ := DefaultPath()
			if got != want {
				t.Fatalf("ResolvePath = %q, want %q", got, want)
			}
		})
	}
}

func TestResolvePathNilLookupUsesProcessEnvironment(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("BEAR_CONFIG", "~/x.toml")

	got, err := ResolvePath("", nil)
	if err != nil {
		t.Fatalf("ResolvePath error = %v", err)
	}
	if want := filepath.Join(home, "x.toml"); got != want {
		t.Fatalf("ResolvePath = %q, want %q", got, want)
	}
}

func TestExpandPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cwd := t.TempDir()
	t.Chdir(cwd)

	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"empty", "", "", true},
		{"tilde alone", "~", home, false},
		{"tilde slash", "~/a/b.toml", filepath.Join(home, "a", "b.toml"), false},
		{"absolute", "/etc/bear.toml", "/etc/bear.toml", false},
		{"relative becomes absolute", "rel.toml", filepath.Join(cwd, "rel.toml"), false},
		{"tilde user is not expanded", "~other/x", filepath.Join(cwd, "~other", "x"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := expandPath(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if normalizeDir(got) != normalizeDir(tt.want) {
				t.Fatalf("expandPath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// normalizeDir resolves symlinks in the parent directory so paths under macOS
// temp dirs (/var -> /private/var) compare equal. Missing parents are left as is.
func normalizeDir(path string) string {
	dir, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return path
	}
	return filepath.Join(dir, filepath.Base(path))
}
