package aws

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeSecretsManager starts an httptest server that stands in for AWS Secrets
// Manager and points the SDK at it via environment variables. The handler
// receives the operation name (X-Amz-Target suffix) and the decoded JSON body.
func fakeSecretsManager(t *testing.T, handler func(op string, body map[string]any, w http.ResponseWriter)) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		op := strings.TrimPrefix(r.Header.Get("X-Amz-Target"), "secretsmanager.")
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		handler(op, body, w)
	}))
	t.Cleanup(srv.Close)

	t.Setenv("AWS_ENDPOINT_URL", srv.URL)
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("AWS_CONFIG_FILE", t.TempDir()+"/config")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", t.TempDir()+"/credentials")
}

func writeAWSError(w http.ResponseWriter, code, msg string) {
	w.Header().Set("X-Amzn-Errortype", code)
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]string{"__type": code, "message": msg})
}

func writeJSON(w http.ResponseWriter, v any) {
	_ = json.NewEncoder(w).Encode(v)
}

func TestCreateSecret_Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   CreateSecretInput
		wantErr string
	}{
		{"empty name", CreateSecretInput{SecretValue: "v"}, "secret name is required"},
		{"blank name", CreateSecretInput{Name: "   ", SecretValue: "v"}, "secret name is required"},
		{"empty value", CreateSecretInput{Name: "n"}, "secret value is required"},
		{"blank value", CreateSecretInput{Name: "n", SecretValue: "  "}, "secret value is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CreateSecret(context.Background(), tt.input)
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("err = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestCreateSecret_Success(t *testing.T) {
	var got map[string]any
	fakeSecretsManager(t, func(op string, body map[string]any, w http.ResponseWriter) {
		if op != "CreateSecret" {
			t.Errorf("unexpected op %q", op)
		}
		got = body
		writeJSON(w, map[string]string{"Name": "my-secret", "ARN": "arn:aws:secretsmanager:::secret:my-secret", "VersionId": "v1"})
	})

	res, err := CreateSecret(context.Background(), CreateSecretInput{
		Name:        "my-secret",
		SecretValue: "s3cr3t",
		Description: "desc",
		Tags:        []string{"env=dev", "team=platform=x", "invalid"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Name != "my-secret" || res.ARN == "" || res.VersionID != "v1" {
		t.Errorf("unexpected result: %+v", res)
	}
	if got["Description"] != "desc" {
		t.Errorf("description not sent: %v", got["Description"])
	}
	tags, _ := got["Tags"].([]any)
	if len(tags) != 2 {
		t.Fatalf("want 2 valid tags (malformed one dropped), got %v", got["Tags"])
	}
	second := tags[1].(map[string]any)
	if second["Key"] != "team" || second["Value"] != "platform=x" {
		t.Errorf("tag should split on first '=' only, got %v", second)
	}
}

func TestCreateSecret_AlreadyExists(t *testing.T) {
	tests := []struct {
		name         string
		override     bool
		wantErr      string
		wantVersion  string
		wantPutValue bool
	}{
		{"without override fails", false, "already exists", "", false},
		{"with override updates value", true, "", "v2", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			putCalled := false
			fakeSecretsManager(t, func(op string, body map[string]any, w http.ResponseWriter) {
				switch op {
				case "CreateSecret":
					writeAWSError(w, "ResourceExistsException", "exists")
				case "PutSecretValue":
					putCalled = true
					writeJSON(w, map[string]string{"ARN": "arn:x", "VersionId": "v2"})
				default:
					t.Errorf("unexpected op %q", op)
				}
			})

			res, err := CreateSecret(context.Background(), CreateSecretInput{
				Name: "dup", SecretValue: "v", Override: tt.override,
			})
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if res.Name != "dup" || res.VersionID != tt.wantVersion {
					t.Errorf("unexpected result: %+v", res)
				}
			}
			if putCalled != tt.wantPutValue {
				t.Errorf("PutSecretValue called = %v, want %v", putCalled, tt.wantPutValue)
			}
		})
	}
}

func TestCreateSecret_OtherError(t *testing.T) {
	fakeSecretsManager(t, func(op string, body map[string]any, w http.ResponseWriter) {
		writeAWSError(w, "InvalidRequestException", "bad")
	})
	_, err := CreateSecret(context.Background(), CreateSecretInput{Name: "n", SecretValue: "v"})
	if err == nil || !strings.Contains(err.Error(), "create secret") {
		t.Fatalf("err = %v, want wrapped 'create secret' error", err)
	}
}

func TestGetSecret_Validation(t *testing.T) {
	_, err := GetSecret(context.Background(), GetSecretInput{Name: " "})
	if err == nil || err.Error() != "secret name is required" {
		t.Fatalf("err = %v", err)
	}
}

func TestGetSecret(t *testing.T) {
	t.Run("string secret", func(t *testing.T) {
		fakeSecretsManager(t, func(op string, body map[string]any, w http.ResponseWriter) {
			if body["SecretId"] != "my-secret" {
				t.Errorf("SecretId = %v", body["SecretId"])
			}
			writeJSON(w, map[string]string{"Name": "my-secret", "SecretString": "val", "ARN": "arn:x", "VersionId": "v1"})
		})
		res, err := GetSecret(context.Background(), GetSecretInput{Name: "my-secret"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.SecretValue != "val" || res.ARN != "arn:x" || res.VersionID != "v1" {
			t.Errorf("unexpected result: %+v", res)
		}
	})

	t.Run("binary secret yields empty value", func(t *testing.T) {
		fakeSecretsManager(t, func(op string, body map[string]any, w http.ResponseWriter) {
			writeJSON(w, map[string]string{"Name": "bin", "SecretBinary": "YWJj"})
		})
		res, err := GetSecret(context.Background(), GetSecretInput{Name: "bin"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.SecretValue != "" {
			t.Errorf("SecretValue = %q, want empty", res.SecretValue)
		}
	})

	t.Run("not found", func(t *testing.T) {
		fakeSecretsManager(t, func(op string, body map[string]any, w http.ResponseWriter) {
			writeAWSError(w, "ResourceNotFoundException", "nope")
		})
		_, err := GetSecret(context.Background(), GetSecretInput{Name: "missing"})
		if err == nil || err.Error() != `secret "missing" not found` {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("other error", func(t *testing.T) {
		fakeSecretsManager(t, func(op string, body map[string]any, w http.ResponseWriter) {
			writeAWSError(w, "AccessDeniedException", "denied")
		})
		_, err := GetSecret(context.Background(), GetSecretInput{Name: "x"})
		if err == nil || !strings.Contains(err.Error(), "get secret") {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestDeleteSecret_Validation(t *testing.T) {
	_, err := DeleteSecret(context.Background(), DeleteSecretInput{})
	if err == nil || err.Error() != "secret name is required" {
		t.Fatalf("err = %v", err)
	}
}

func TestDeleteSecret_RequestShape(t *testing.T) {
	tests := []struct {
		name         string
		input        DeleteSecretInput
		wantForce    any
		wantRecovery any
	}{
		{"force delete", DeleteSecretInput{Name: "s", ForceDelete: true, RecoveryDays: 10}, true, nil},
		{"recovery window", DeleteSecretInput{Name: "s", RecoveryDays: 14}, nil, float64(14)},
		{"defaults", DeleteSecretInput{Name: "s"}, nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got map[string]any
			fakeSecretsManager(t, func(op string, body map[string]any, w http.ResponseWriter) {
				got = body
				writeJSON(w, map[string]any{"Name": "s", "ARN": "arn:x", "DeletionDate": 1700000000})
			})
			res, err := DeleteSecret(context.Background(), tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.Name != "s" || res.ARN != "arn:x" || res.DeletionDate == "" {
				t.Errorf("unexpected result: %+v", res)
			}
			if got["ForceDeleteWithoutRecovery"] != tt.wantForce {
				t.Errorf("ForceDeleteWithoutRecovery = %v, want %v", got["ForceDeleteWithoutRecovery"], tt.wantForce)
			}
			if got["RecoveryWindowInDays"] != tt.wantRecovery {
				t.Errorf("RecoveryWindowInDays = %v, want %v", got["RecoveryWindowInDays"], tt.wantRecovery)
			}
		})
	}
}

func TestDeleteSecret_Errors(t *testing.T) {
	tests := []struct {
		code    string
		wantErr string
	}{
		{"ResourceNotFoundException", `secret "s" not found`},
		{"AccessDeniedException", "delete secret"},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			fakeSecretsManager(t, func(op string, body map[string]any, w http.ResponseWriter) {
				writeAWSError(w, tt.code, "x")
			})
			_, err := DeleteSecret(context.Background(), DeleteSecretInput{Name: "s"})
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}
