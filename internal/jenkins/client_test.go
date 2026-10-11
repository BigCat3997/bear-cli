package jenkins

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestCreateSecretTextCredentials(t *testing.T) {
	var bodies []string
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != defaultCredentialsPath {
			t.Errorf("path = %q", r.URL.Path)
		}
		username, token, ok := r.BasicAuth()
		if !ok || username != "admin" || token != "token" {
			t.Errorf("unexpected authentication: %q %q %v", username, token, ok)
		}
		if got := r.Header.Get("Content-Type"); got != "application/xml" {
			t.Errorf("content type = %q", got)
		}
		body, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(body))
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}, nil
	})

	client := NewClient("https://jenkins.test", "admin", "token")
	client.HTTPClient = &http.Client{Transport: transport}
	created, err := client.CreateSecretTextCredentials(context.Background(), map[string]string{
		"AWS_SECRET_ACCESS_KEY": "secret<&>",
		"AWS_REGION":            "us-east-1",
	}, PublishOptions{Scope: "global", IDPrefix: "PS AWS"})
	if err != nil {
		t.Fatalf("CreateSecretTextCredentials: %v", err)
	}

	want := []CreatedCredential{
		{ID: "ps-aws-aws-region", Variable: "AWS_REGION", Scope: "GLOBAL", Status: "created"},
		{ID: "ps-aws-aws-secret-access-key", Variable: "AWS_SECRET_ACCESS_KEY", Scope: "GLOBAL", Status: "created"},
	}
	if !reflect.DeepEqual(created, want) {
		t.Fatalf("created = %#v, want %#v", created, want)
	}
	if len(bodies) != 2 {
		t.Fatalf("request count = %d, want 2", len(bodies))
	}
	if !strings.Contains(bodies[1], "<secret>secret&lt;&amp;&gt;</secret>") {
		t.Fatalf("secret was not XML escaped: %s", bodies[1])
	}
}

func TestCreateSecretTextCredentialsOverridesExistingCredential(t *testing.T) {
	var requestPath string
	client := NewClient("https://jenkins.test", "admin", "token")
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requestPath = request.URL.Path
		return response(http.StatusOK, ""), nil
	})}

	created, err := client.CreateSecretTextCredentials(context.Background(), map[string]string{"AWS_REGION": "ap-southeast-1"}, PublishOptions{
		Scope: "GLOBAL", IDPrefix: "ps-aws", Override: true,
	})
	if err != nil {
		t.Fatalf("CreateSecretTextCredentials: %v", err)
	}
	if want := "/credentials/store/system/domain/_/credential/ps-aws-aws-region/config.xml"; requestPath != want {
		t.Fatalf("path = %q, want %q", requestPath, want)
	}
	if len(created) != 1 || created[0].Status != "updated" {
		t.Fatalf("created = %#v", created)
	}
}

func TestCreateSecretTextCredentialsOverrideCreatesMissingCredential(t *testing.T) {
	var paths []string
	client := NewClient("https://jenkins.test", "admin", "token")
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		paths = append(paths, request.URL.Path)
		if len(paths) == 1 {
			return response(http.StatusNotFound, "missing"), nil
		}
		return response(http.StatusOK, ""), nil
	})}

	created, err := client.CreateSecretTextCredentials(context.Background(), map[string]string{"AWS_REGION": "us-east-1"}, PublishOptions{
		Scope: "GLOBAL", IDPrefix: "ps-aws", Override: true,
	})
	if err != nil {
		t.Fatalf("CreateSecretTextCredentials: %v", err)
	}
	if len(paths) != 2 || paths[1] != defaultCredentialsPath {
		t.Fatalf("paths = %v", paths)
	}
	if len(created) != 1 || created[0].Status != "created" {
		t.Fatalf("created = %#v", created)
	}
}

func TestCreateSecretTextCredentialsRejectsInvalidScopeBeforeRequest(t *testing.T) {
	client := NewClient("https://jenkins.example", "admin", "token")
	_, err := client.CreateSecretTextCredentials(context.Background(), map[string]string{"KEY": "value"}, PublishOptions{Scope: "folder"})
	if err == nil || !strings.Contains(err.Error(), "GLOBAL or SYSTEM") {
		t.Fatalf("error = %v", err)
	}
}

func TestPutSecretTextCredentialKeepsExactIDAndEscapesSecret(t *testing.T) {
	var body string
	client := NewClient("https://jenkins.test", "admin", "token")
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		content, _ := io.ReadAll(request.Body)
		body = string(content)
		return response(http.StatusOK, ""), nil
	})}

	result, err := client.PutSecretTextCredential(context.Background(), SecretTextCredentialInput{
		ID: "Deploy.Token", Secret: "secret<&>", Scope: "system",
	})
	if err != nil {
		t.Fatalf("PutSecretTextCredential: %v", err)
	}
	if result.ID != "Deploy.Token" || result.Scope != "SYSTEM" || result.Status != "created" {
		t.Fatalf("result = %#v", result)
	}
	if !strings.Contains(body, "<id>Deploy.Token</id>") || !strings.Contains(body, "<secret>secret&lt;&amp;&gt;</secret>") {
		t.Fatalf("unexpected credential XML: %s", body)
	}
}

func TestPutSSHPrivateKeyCredentialPostsJenkinsCredentialXML(t *testing.T) {
	var body string
	client := NewClient("https://jenkins.test", "admin", "token")
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != defaultCredentialsPath {
			t.Errorf("path = %q", request.URL.Path)
		}
		content, _ := io.ReadAll(request.Body)
		body = string(content)
		return response(http.StatusOK, ""), nil
	})}

	result, err := client.PutSSHPrivateKeyCredential(context.Background(), SSHPrivateKeyCredentialInput{
		ID: "ps-agent-1", Username: "ubuntu", PrivateKey: "-----BEGIN PRIVATE KEY-----\nsecret<&>\n-----END PRIVATE KEY-----\n",
		Passphrase: "pass<&>", Description: "Linux agent", Scope: "system",
	})
	if err != nil {
		t.Fatalf("PutSSHPrivateKeyCredential: %v", err)
	}
	if result.ID != "ps-agent-1" || result.Scope != "SYSTEM" || result.Status != "created" {
		t.Fatalf("result = %#v", result)
	}
	for _, fragment := range []string{
		"<com.cloudbees.jenkins.plugins.sshcredentials.impl.BasicSSHUserPrivateKey>",
		"<scope>SYSTEM</scope>", "<id>ps-agent-1</id>", "<username>ubuntu</username>",
		"<usernameSecret>false</usernameSecret>",
		`class="com.cloudbees.jenkins.plugins.sshcredentials.impl.BasicSSHUserPrivateKey$DirectEntryPrivateKeySource"`,
		"secret&lt;&amp;&gt;", "<passphrase>pass&lt;&amp;&gt;</passphrase>",
	} {
		if !strings.Contains(body, fragment) {
			t.Errorf("credential XML missing %q: %s", fragment, body)
		}
	}
}

func TestPutSSHPrivateKeyCredentialValidatesRequiredFields(t *testing.T) {
	tests := []struct {
		name  string
		input SSHPrivateKeyCredentialInput
		want  string
	}{
		{name: "id", input: SSHPrivateKeyCredentialInput{Username: "ubuntu", PrivateKey: "key"}, want: "credential ID"},
		{name: "username", input: SSHPrivateKeyCredentialInput{ID: "key", PrivateKey: "key"}, want: "SSH username"},
		{name: "private key", input: SSHPrivateKeyCredentialInput{ID: "key", Username: "ubuntu"}, want: "private key"},
	}

	client := NewClient("https://jenkins.test", "admin", "token")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.PutSSHPrivateKeyCredential(context.Background(), tt.input)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.want)
			}
		})
	}
}

func TestCreateSSHAgentPostsXML(t *testing.T) {
	var body string
	client := NewClient("https://jenkins.test", "admin", "token")
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != createAgentPath {
			t.Errorf("path = %q", request.URL.Path)
		}
		if got := request.URL.Query().Get("name"); got != "linux agent/01" {
			t.Errorf("name query = %q", got)
		}
		if got := request.Header.Get("Content-Type"); got != "application/xml" {
			t.Errorf("content type = %q", got)
		}
		content, _ := io.ReadAll(request.Body)
		body = string(content)
		return response(http.StatusOK, ""), nil
	})}

	result, err := client.CreateSSHAgent(context.Background(), validSSHAgentInput())
	if err != nil {
		t.Fatalf("CreateSSHAgent: %v", err)
	}
	if result.Name != "linux agent/01" || result.Status != "created" {
		t.Fatalf("result = %#v", result)
	}
	for _, fragment := range []string{
		"<slave>", "<remoteFS>/home/jenkins &amp; jobs</remoteFS>", "<numExecutors>2</numExecutors>",
		"<mode>EXCLUSIVE</mode>", `class="hudson.plugins.sshslaves.SSHLauncher"`,
		"<credentialsId>agent-key</credentialsId>", "KnownHostsFileKeyVerificationStrategy",
	} {
		if !strings.Contains(body, fragment) {
			t.Errorf("agent XML missing %q: %s", fragment, body)
		}
	}
}

func TestCreateSSHAgentOverrideUpdatesExistingAgent(t *testing.T) {
	var paths []string
	client := NewClient("https://jenkins.test", "admin", "token")
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		paths = append(paths, request.URL.EscapedPath())
		return response(http.StatusOK, ""), nil
	})}

	input := validSSHAgentInput()
	input.Override = true
	result, err := client.CreateSSHAgent(context.Background(), input)
	if err != nil {
		t.Fatalf("CreateSSHAgent: %v", err)
	}
	if result.Status != "updated" {
		t.Fatalf("result = %#v", result)
	}
	if !reflect.DeepEqual(paths, []string{"/computer/linux%20agent%2F01/config.xml"}) {
		t.Fatalf("paths = %v", paths)
	}
}

func TestCreateSSHAgentOverrideCreatesMissingAgent(t *testing.T) {
	var paths []string
	client := NewClient("https://jenkins.test", "admin", "token")
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		paths = append(paths, request.URL.Path)
		if len(paths) == 1 {
			return response(http.StatusNotFound, "missing"), nil
		}
		return response(http.StatusOK, ""), nil
	})}

	input := validSSHAgentInput()
	input.Override = true
	result, err := client.CreateSSHAgent(context.Background(), input)
	if err != nil {
		t.Fatalf("CreateSSHAgent: %v", err)
	}
	if result.Status != "created" || !reflect.DeepEqual(paths, []string{"/computer/linux agent/01/config.xml", createAgentPath}) {
		t.Fatalf("result = %#v, paths = %v", result, paths)
	}
}

func TestCreateSSHAgentRejectsUnsafeDefaults(t *testing.T) {
	input := validSSHAgentInput()
	input.HostKeyVerificationStrategy = "trust-everything"
	client := NewClient("https://jenkins.test", "admin", "token")
	_, err := client.CreateSSHAgent(context.Background(), input)
	if err == nil || !strings.Contains(err.Error(), "host key strategy") {
		t.Fatalf("error = %v", err)
	}
}

func validSSHAgentInput() SSHAgentInput {
	return SSHAgentInput{
		Name: "linux agent/01", Host: "agent.example", Port: 22, CredentialsID: "agent-key",
		RemoteFS: "/home/jenkins & jobs", Executors: 2, Mode: "exclusive", Labels: "linux terraform",
		LaunchTimeoutSeconds: 60, MaxNumRetries: 10, RetryWaitTimeSeconds: 15,
		HostKeyVerificationStrategy: "known-hosts",
	}
}

func TestSlug(t *testing.T) {
	if got, want := slug(" ARM_CLIENT_ID "), "arm-client-id"; got != want {
		t.Fatalf("slug = %q, want %q", got, want)
	}
}

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestCreateSecretTextCredentialsWithoutPrefixUsesVariableName(t *testing.T) {
	var requestPath string
	client := NewClient("https://jenkins.test", "admin", "token")
	client.HTTPClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requestPath = request.URL.Path
		return response(http.StatusOK, ""), nil
	})}

	created, err := client.CreateSecretTextCredentials(context.Background(), map[string]string{"AWS_ACCESS_KEY_ID": "id"}, PublishOptions{Scope: "GLOBAL"})
	if err != nil {
		t.Fatalf("CreateSecretTextCredentials: %v", err)
	}
	if len(created) != 1 || created[0].ID != "aws-access-key-id" {
		t.Fatalf("created = %#v, want ID aws-access-key-id", created)
	}
	if want := "/credentials/store/system/domain/_/credential/aws-access-key-id/config.xml"; requestPath != want && requestPath != "/credentials/store/system/domain/_/createCredentials" {
		t.Fatalf("path = %q", requestPath)
	}
}
