package jenkins

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const defaultCredentialsPath = "/credentials/store/system/domain/_/createCredentials"

const credentialConfigPath = "/credentials/store/system/domain/_/credential/%s/config.xml"

const createAgentPath = "/computer/doCreateItem"

const agentConfigPath = "/computer/%s/config.xml"

type Client struct {
	BaseURL    string
	Username   string
	APIToken   string
	HTTPClient *http.Client
}

type PublishOptions struct {
	Scope       string
	IDPrefix    string
	Description string
	Override    bool
}

type CreatedCredential struct {
	ID       string `json:"id"`
	Variable string `json:"variable"`
	Scope    string `json:"scope"`
	Status   string `json:"status"`
}

type SecretTextCredentialInput struct {
	ID          string
	Secret      string
	Description string
	Scope       string
	Override    bool
}

type SSHPrivateKeyCredentialInput struct {
	ID          string
	Username    string
	PrivateKey  string
	Passphrase  string
	Description string
	Scope       string
	Override    bool
}

type SSHAgentInput struct {
	Name                        string
	Description                 string
	RemoteFS                    string
	Labels                      string
	Host                        string
	CredentialsID               string
	Executors                   int
	Mode                        string
	Port                        int
	JavaPath                    string
	JVMOptions                  string
	PrefixStartAgentCommand     string
	SuffixStartAgentCommand     string
	LaunchTimeoutSeconds        int
	MaxNumRetries               int
	RetryWaitTimeSeconds        int
	HostKeyVerificationStrategy string
	Override                    bool
}

type CreatedAgent struct {
	Name   string `json:"name"`
	Host   string `json:"host"`
	Status string `json:"status"`
}

type stringCredentialXML struct {
	XMLName     xml.Name `xml:"org.jenkinsci.plugins.plaincredentials.impl.StringCredentialsImpl"`
	Scope       string   `xml:"scope"`
	ID          string   `xml:"id"`
	Description string   `xml:"description"`
	Secret      string   `xml:"secret"`
}

type sshPrivateKeyCredentialXML struct {
	XMLName          xml.Name            `xml:"com.cloudbees.jenkins.plugins.sshcredentials.impl.BasicSSHUserPrivateKey"`
	Scope            string              `xml:"scope"`
	ID               string              `xml:"id"`
	Description      string              `xml:"description"`
	Username         string              `xml:"username"`
	UsernameSecret   bool                `xml:"usernameSecret"`
	PrivateKeySource directPrivateKeyXML `xml:"privateKeySource"`
	Passphrase       string              `xml:"passphrase,omitempty"`
}

type directPrivateKeyXML struct {
	Class      string `xml:"class,attr"`
	PrivateKey string `xml:"privateKey"`
}

type sshAgentXML struct {
	XMLName        xml.Name       `xml:"slave"`
	Name           string         `xml:"name"`
	Description    string         `xml:"nodeDescription"`
	RemoteFS       string         `xml:"remoteFS"`
	Executors      int            `xml:"numExecutors"`
	Mode           string         `xml:"mode"`
	Retention      classXML       `xml:"retentionStrategy"`
	Launcher       sshLauncherXML `xml:"launcher"`
	Labels         string         `xml:"label"`
	NodeProperties struct{}       `xml:"nodeProperties"`
}

type classXML struct {
	Class string `xml:"class,attr"`
}

type sshLauncherXML struct {
	Class                   string             `xml:"class,attr"`
	Plugin                  string             `xml:"plugin,attr"`
	Host                    string             `xml:"host"`
	Port                    int                `xml:"port"`
	CredentialsID           string             `xml:"credentialsId"`
	JVMOptions              string             `xml:"jvmOptions"`
	JavaPath                string             `xml:"javaPath"`
	PrefixStartAgentCommand string             `xml:"prefixStartSlaveCmd"`
	SuffixStartAgentCommand string             `xml:"suffixStartSlaveCmd"`
	LaunchTimeoutSeconds    int                `xml:"launchTimeoutSeconds"`
	MaxNumRetries           int                `xml:"maxNumRetries"`
	RetryWaitTimeSeconds    int                `xml:"retryWaitTime"`
	HostKeyStrategy         hostKeyStrategyXML `xml:"sshHostKeyVerificationStrategy"`
	TCPNoDelay              bool               `xml:"tcpNoDelay"`
	TrackCredentials        bool               `xml:"trackCredentials"`
}

type hostKeyStrategyXML struct {
	Class                     string `xml:"class,attr"`
	RequireInitialManualTrust *bool  `xml:"requireInitialManualTrust,omitempty"`
}

func NewClient(baseURL, username, apiToken string) *Client {
	return &Client{
		BaseURL:  strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		Username: strings.TrimSpace(username),
		APIToken: apiToken,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// PutSecretTextCredential creates a single Jenkins Secret Text credential. If
// Override is true, an existing credential with the same ID is updated.
func (c *Client) PutSecretTextCredential(ctx context.Context, input SecretTextCredentialInput) (*CreatedCredential, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}

	input.ID = strings.TrimSpace(input.ID)
	if input.ID == "" {
		return nil, fmt.Errorf("Jenkins credential ID is required")
	}
	scope, err := normalizeCredentialScope(input.Scope)
	if err != nil {
		return nil, err
	}

	payload, err := xml.Marshal(stringCredentialXML{
		Scope:       scope,
		ID:          input.ID,
		Description: strings.TrimSpace(input.Description),
		Secret:      input.Secret,
	})
	if err != nil {
		return nil, fmt.Errorf("encode Jenkins credential %q: %w", input.ID, err)
	}

	status, err := c.putCredential(ctx, input.ID, payload, input.Override)
	if err != nil {
		return nil, fmt.Errorf("publish Jenkins credential %q: %w", input.ID, err)
	}
	return &CreatedCredential{ID: input.ID, Scope: scope, Status: status}, nil
}

// PutSSHPrivateKeyCredential creates a Jenkins "SSH Username with private key"
// credential. The Jenkins controller must have the SSH Credentials plugin.
func (c *Client) PutSSHPrivateKeyCredential(ctx context.Context, input SSHPrivateKeyCredentialInput) (*CreatedCredential, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}

	input.ID = strings.TrimSpace(input.ID)
	input.Username = strings.TrimSpace(input.Username)
	if input.ID == "" {
		return nil, fmt.Errorf("Jenkins credential ID is required")
	}
	if input.Username == "" {
		return nil, fmt.Errorf("SSH username is required")
	}
	if strings.TrimSpace(input.PrivateKey) == "" {
		return nil, fmt.Errorf("SSH private key is required")
	}
	scope, err := normalizeCredentialScope(input.Scope)
	if err != nil {
		return nil, err
	}

	payload, err := xml.Marshal(sshPrivateKeyCredentialXML{
		Scope:          scope,
		ID:             input.ID,
		Description:    strings.TrimSpace(input.Description),
		Username:       input.Username,
		UsernameSecret: false,
		PrivateKeySource: directPrivateKeyXML{
			Class:      "com.cloudbees.jenkins.plugins.sshcredentials.impl.BasicSSHUserPrivateKey$DirectEntryPrivateKeySource",
			PrivateKey: input.PrivateKey,
		},
		Passphrase: input.Passphrase,
	})
	if err != nil {
		return nil, fmt.Errorf("encode Jenkins SSH private-key credential %q: %w", input.ID, err)
	}

	status, err := c.putCredential(ctx, input.ID, payload, input.Override)
	if err != nil {
		return nil, fmt.Errorf("publish Jenkins SSH private-key credential %q: %w", input.ID, err)
	}
	return &CreatedCredential{ID: input.ID, Scope: scope, Status: status}, nil
}

// CreateSSHAgent creates a permanent Jenkins agent launched over SSH. Jenkins
// must have the SSH Build Agents plugin and CredentialsID must refer to a
// Jenkins username/password or username/private-key credential.
func (c *Client) CreateSSHAgent(ctx context.Context, input SSHAgentInput) (*CreatedAgent, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}
	if err := normalizeSSHAgentInput(&input); err != nil {
		return nil, err
	}

	payload, err := marshalSSHAgentXML(input)
	if err != nil {
		return nil, fmt.Errorf("encode Jenkins SSH agent %q: %w", input.Name, err)
	}

	status := "created"
	if input.Override {
		configPath := fmt.Sprintf(agentConfigPath, url.PathEscape(input.Name))
		notFound := http.StatusNotFound
		if err := c.sendXML(ctx, configPath, payload, &notFound); err != nil {
			return nil, fmt.Errorf("update Jenkins SSH agent %q: %w", input.Name, err)
		}
		if notFound == 0 {
			status = "updated"
			return &CreatedAgent{Name: input.Name, Host: input.Host, Status: status}, nil
		}
	}

	query := url.Values{"name": []string{input.Name}}
	if err := c.sendXML(ctx, createAgentPath+"?"+query.Encode(), payload, nil); err != nil {
		return nil, fmt.Errorf("create Jenkins SSH agent %q: %w", input.Name, err)
	}
	return &CreatedAgent{Name: input.Name, Host: input.Host, Status: status}, nil
}

// CreateSecretTextCredentials publishes each environment variable as a
// Jenkins Secret Text credential. This representation works with the Jenkins
// Credentials Binding and Plain Credentials plugins and avoids writing secrets
// to command output.
func (c *Client) CreateSecretTextCredentials(ctx context.Context, values map[string]string, options PublishOptions) ([]CreatedCredential, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("at least one credential value is required")
	}

	scope, err := normalizeCredentialScope(options.Scope)
	if err != nil {
		return nil, err
	}

	prefix := slug(options.IDPrefix)
	if prefix == "" {
		prefix = "bear"
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	created := make([]CreatedCredential, 0, len(keys))
	for _, key := range keys {
		credentialID := prefix + "-" + slug(key)
		description := strings.TrimSpace(options.Description)
		if description == "" {
			description = "Bear-managed credential for " + key
		}

		result, err := c.PutSecretTextCredential(ctx, SecretTextCredentialInput{
			ID: credentialID, Secret: values[key], Description: description, Scope: scope, Override: options.Override,
		})
		if err != nil {
			return created, err
		}
		result.Variable = key
		created = append(created, *result)
	}

	return created, nil
}

func normalizeCredentialScope(value string) (string, error) {
	scope := strings.ToUpper(strings.TrimSpace(value))
	if scope == "" {
		scope = "GLOBAL"
	}
	if scope != "GLOBAL" && scope != "SYSTEM" {
		return "", fmt.Errorf("invalid Jenkins credential scope %q: expected GLOBAL or SYSTEM", value)
	}
	return scope, nil
}

func normalizeSSHAgentInput(input *SSHAgentInput) error {
	input.Name = strings.TrimSpace(input.Name)
	input.Host = strings.TrimSpace(input.Host)
	input.RemoteFS = strings.TrimSpace(input.RemoteFS)
	input.CredentialsID = strings.TrimSpace(input.CredentialsID)
	input.Mode = strings.ToUpper(strings.TrimSpace(input.Mode))
	input.HostKeyVerificationStrategy = strings.ToLower(strings.TrimSpace(input.HostKeyVerificationStrategy))
	if input.Name == "" || input.Host == "" || input.RemoteFS == "" || input.CredentialsID == "" {
		return fmt.Errorf("Jenkins SSH agent name, host, remote FS, and credentials ID are required")
	}
	if input.Executors <= 0 {
		return fmt.Errorf("Jenkins SSH agent executors must be greater than zero")
	}
	if input.Port <= 0 || input.Port > 65535 {
		return fmt.Errorf("invalid SSH port %d", input.Port)
	}
	if input.Mode == "" {
		input.Mode = "NORMAL"
	}
	if input.Mode != "NORMAL" && input.Mode != "EXCLUSIVE" {
		return fmt.Errorf("invalid Jenkins agent mode %q: expected NORMAL or EXCLUSIVE", input.Mode)
	}
	if input.LaunchTimeoutSeconds <= 0 || input.MaxNumRetries < 0 || input.RetryWaitTimeSeconds < 0 {
		return fmt.Errorf("launch timeout must be positive and retry values cannot be negative")
	}
	if input.HostKeyVerificationStrategy == "" {
		input.HostKeyVerificationStrategy = "known-hosts"
	}
	if _, _, err := hostKeyStrategy(input.HostKeyVerificationStrategy); err != nil {
		return err
	}
	return nil
}

func marshalSSHAgentXML(input SSHAgentInput) ([]byte, error) {
	class, manualTrust, err := hostKeyStrategy(input.HostKeyVerificationStrategy)
	if err != nil {
		return nil, err
	}
	return xml.Marshal(sshAgentXML{
		Name: input.Name, Description: strings.TrimSpace(input.Description), RemoteFS: input.RemoteFS,
		Executors: input.Executors, Mode: input.Mode, Labels: strings.TrimSpace(input.Labels),
		Retention: classXML{Class: "hudson.slaves.RetentionStrategy$Always"},
		Launcher: sshLauncherXML{
			Class: "hudson.plugins.sshslaves.SSHLauncher", Plugin: "ssh-slaves", Host: input.Host,
			Port: input.Port, CredentialsID: input.CredentialsID, JVMOptions: input.JVMOptions,
			JavaPath: input.JavaPath, PrefixStartAgentCommand: input.PrefixStartAgentCommand,
			SuffixStartAgentCommand: input.SuffixStartAgentCommand, LaunchTimeoutSeconds: input.LaunchTimeoutSeconds,
			MaxNumRetries: input.MaxNumRetries, RetryWaitTimeSeconds: input.RetryWaitTimeSeconds,
			HostKeyStrategy: hostKeyStrategyXML{Class: class, RequireInitialManualTrust: manualTrust},
			TCPNoDelay:      true, TrackCredentials: true,
		},
	})
}

func hostKeyStrategy(value string) (string, *bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "known-hosts":
		return "hudson.plugins.sshslaves.verifiers.KnownHostsFileKeyVerificationStrategy", nil, nil
	case "manually-trusted":
		requireTrust := true
		return "hudson.plugins.sshslaves.verifiers.ManuallyTrustedKeyVerificationStrategy", &requireTrust, nil
	case "non-verifying":
		return "hudson.plugins.sshslaves.verifiers.NonVerifyingKeyVerificationStrategy", nil, nil
	default:
		return "", nil, fmt.Errorf("invalid host key strategy %q: expected known-hosts, manually-trusted, or non-verifying", value)
	}
}

func (c *Client) validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("Jenkins URL is required")
	}
	parsed, err := url.Parse(c.BaseURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("invalid Jenkins URL %q: expected an http or https URL", c.BaseURL)
	}
	if c.Username == "" {
		return fmt.Errorf("Jenkins username is required")
	}
	if strings.TrimSpace(c.APIToken) == "" {
		return fmt.Errorf("Jenkins API token is required")
	}
	if c.HTTPClient == nil {
		return fmt.Errorf("Jenkins HTTP client is required")
	}
	return nil
}

func (c *Client) putCredential(ctx context.Context, credentialID string, payload []byte, override bool) (string, error) {
	if !override {
		if err := c.sendCredentialXML(ctx, defaultCredentialsPath, payload, nil); err != nil {
			return "", err
		}
		return "created", nil
	}

	updatePath := fmt.Sprintf(credentialConfigPath, url.PathEscape(credentialID))
	notFound := http.StatusNotFound
	if err := c.sendCredentialXML(ctx, updatePath, payload, &notFound); err != nil {
		return "", err
	}
	if notFound == 0 {
		return "updated", nil
	}

	if err := c.sendCredentialXML(ctx, defaultCredentialsPath, payload, nil); err != nil {
		return "", err
	}
	return "created", nil
}

func (c *Client) sendCredentialXML(ctx context.Context, path string, payload []byte, allowedStatus *int) error {
	return c.sendXML(ctx, path, payload, allowedStatus)
}

func (c *Client) sendXML(ctx context.Context, path string, payload []byte, allowedStatus *int) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.Username, c.APIToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/xml")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		if allowedStatus != nil {
			*allowedStatus = 0
		}
		return nil
	}
	if allowedStatus != nil && resp.StatusCode == *allowedStatus {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = resp.Status
	}
	return fmt.Errorf("Jenkins returned %s: %s", resp.Status, message)
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var result strings.Builder
	lastDash := false
	for _, char := range value {
		valid := (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')
		if valid {
			result.WriteRune(char)
			lastDash = false
			continue
		}
		if result.Len() > 0 && !lastDash {
			result.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(result.String(), "-")
}
