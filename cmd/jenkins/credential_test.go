package jenkins

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveSecretTextFromEnvironment(t *testing.T) {
	t.Setenv("BEAR_TEST_JENKINS_SECRET", "secret-value")
	got, err := resolveSecretText(&secretTextOptions{SecretEnv: "BEAR_TEST_JENKINS_SECRET"})
	if err != nil {
		t.Fatalf("resolveSecretText: %v", err)
	}
	if got != "secret-value" {
		t.Fatalf("secret = %q", got)
	}
}

func TestResolveSecretTextFromFileRemovesOneTrailingNewline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(path, []byte("secret-value\r\n"), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}
	got, err := resolveSecretText(&secretTextOptions{SecretFile: path})
	if err != nil {
		t.Fatalf("resolveSecretText: %v", err)
	}
	if got != "secret-value" {
		t.Fatalf("secret = %q", got)
	}
}

func TestResolveSecretTextRejectsMissingEnvironmentVariable(t *testing.T) {
	_, err := resolveSecretText(&secretTextOptions{SecretEnv: "BEAR_TEST_MISSING_JENKINS_SECRET"})
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestResolvePrivateKeyFromFilePreservesPEM(t *testing.T) {
	path := filepath.Join(t.TempDir(), "id_ed25519")
	want := "-----BEGIN PRIVATE KEY-----\nprivate-data\n-----END PRIVATE KEY-----\n"
	if err := os.WriteFile(path, []byte(want), 0o600); err != nil {
		t.Fatalf("write private key: %v", err)
	}

	got, err := resolveFileOrEnvironment(path, "", "private key")
	if err != nil {
		t.Fatalf("resolveFileOrEnvironment: %v", err)
	}
	if got != want {
		t.Fatalf("private key = %q, want %q", got, want)
	}
}

func TestResolvePrivateKeyFromEnvironment(t *testing.T) {
	t.Setenv("BEAR_TEST_PRIVATE_KEY", "private-key")
	got, err := resolveFileOrEnvironment("", "BEAR_TEST_PRIVATE_KEY", "private key")
	if err != nil {
		t.Fatalf("resolveFileOrEnvironment: %v", err)
	}
	if got != "private-key" {
		t.Fatalf("private key = %q", got)
	}
}

func TestSSHPrivateKeyCommandIsRegistered(t *testing.T) {
	cmd := newCredentialCommand(&connectionOptions{})
	found, _, err := cmd.Find([]string{"create-ssh-private-key"})
	if err != nil || found == cmd || found.Name() != "create-ssh-private-key" {
		t.Fatalf("create-ssh-private-key command not registered: found=%v err=%v", found, err)
	}
}
