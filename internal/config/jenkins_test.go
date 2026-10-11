package config

import (
	"os"
	"path/filepath"
	"testing"
)

const jenkinsTestConfig = `
[jenkins]
jenkins_url = "http://from-file:8080/"
jenkins_user_id = "file-user"
jenkins_api_token = "file-token"
`

func writeJenkinsTestConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(jenkinsTestConfig), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func emptyEnvironment(string) (string, bool) {
	return "", false
}

func TestLoadJenkinsConfigFromFile(t *testing.T) {
	path := writeJenkinsTestConfig(t)
	result, err := Load(LoadOptions{Path: path, LookupEnv: emptyEnvironment})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := JenkinsConfig{URL: "http://from-file:8080", UserID: "file-user", APIToken: "file-token"}
	if result.Config.Jenkins != want {
		t.Fatalf("Jenkins config = %#v, want %#v", result.Config.Jenkins, want)
	}
}

func TestBearJenkinsEnvironmentOverridesFile(t *testing.T) {
	path := writeJenkinsTestConfig(t)
	environment := map[string]string{
		"BEAR_JENKINS_URL":       "http://from-env:9090/",
		"BEAR_JENKINS_USER_ID":   "env-user",
		"BEAR_JENKINS_API_TOKEN": "env-token",
	}
	result, err := Load(LoadOptions{Path: path, LookupEnv: func(key string) (string, bool) {
		value, found := environment[key]
		return value, found
	}})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := JenkinsConfig{URL: "http://from-env:9090", UserID: "env-user", APIToken: "env-token"}
	if result.Config.Jenkins != want {
		t.Fatalf("Jenkins config = %#v, want %#v", result.Config.Jenkins, want)
	}
}

func TestEmptyBearJenkinsEnvironmentFallsBackToFile(t *testing.T) {
	path := writeJenkinsTestConfig(t)
	environment := map[string]string{
		"BEAR_JENKINS_URL":       "",
		"BEAR_JENKINS_USER_ID":   "   ",
		"BEAR_JENKINS_API_TOKEN": "",
	}
	result, err := Load(LoadOptions{Path: path, LookupEnv: func(key string) (string, bool) {
		value, found := environment[key]
		return value, found
	}})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := JenkinsConfig{URL: "http://from-file:8080", UserID: "file-user", APIToken: "file-token"}
	if result.Config.Jenkins != want {
		t.Fatalf("Jenkins config = %#v, want %#v", result.Config.Jenkins, want)
	}
}
