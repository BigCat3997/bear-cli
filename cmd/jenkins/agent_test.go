package jenkins

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveSSHAgentConnectionLoadsMissingValuesFromConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	contents := []byte(`[jenkins]
jenkins_url = "http://from-file:8080/"
jenkins_user_id = "file-user"
jenkins_api_token = "file-token"
`)
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv("BEAR_CONFIG", path)
	t.Setenv("BEAR_JENKINS_URL", "")
	t.Setenv("BEAR_JENKINS_USER_ID", "")
	t.Setenv("BEAR_JENKINS_API_TOKEN", "")

	got, err := resolveSSHAgentConnection(&connectionOptions{URL: "http://from-flag:9090/"})
	if err != nil {
		t.Fatalf("resolveSSHAgentConnection: %v", err)
	}

	if got.URL != "http://from-flag:9090/" {
		t.Fatalf("URL = %q, want explicit flag value", got.URL)
	}
	if got.UserID != "file-user" {
		t.Fatalf("UserID = %q, want file-user", got.UserID)
	}
	if got.APIToken != "file-token" {
		t.Fatalf("APIToken = %q, want file-token", got.APIToken)
	}
}
