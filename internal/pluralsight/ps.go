package ps

import (
	"bear_cli/internal/arm"
	"bear_cli/internal/browser"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/atotto/clipboard"
)

type Command string

const (
	Ps                  Command = "ps"
	PsCreateCredential  Command = "create"
	PsGetCredential     Command = "get"
	PsInitCredential    Command = "init"
	PsTerraform         Command = "terraform"
	PsLoginByCredential Command = "login"
)

var CommandDescriptions = map[Command]string{
	Ps:                  "Interact with PluralSight resources",
	PsCreateCredential:  "Create credential",
	PsGetCredential:     "Get credential",
	PsInitCredential:    "Initialize credential in every consumed environment",
	PsTerraform:         "Set up Terraform backend resources",
	PsLoginByCredential: "Log in to appropriate cloud by credential",
}

type TerraformSetup struct {
	BackendType  string            `json:"backendType"`
	ResourceType string            `json:"resourceType"`
	Environment  map[string]string `json:"environment"`
}

func RunExtractor(useClipboard bool, htmlPath string, cloudProvider CloudProvider) map[string]string {
	creds, err := ExtractCredential(useClipboard, htmlPath, cloudProvider)
	if err != nil {
		log.Fatal(err)
	}
	return creds
}

// ExtractCredential reads and parses a Pluralsight sandbox credential page.
// Unlike RunExtractor it returns errors to callers that must not terminate the
// entire process.
func ExtractCredential(useClipboard bool, htmlPath string, cloudProvider CloudProvider) (map[string]string, error) {
	var htmlContent string
	var err error

	if useClipboard {
		htmlContent, err = clipboard.ReadAll()
		if err != nil {
			return nil, fmt.Errorf("read clipboard: %w", err)
		}
	} else if htmlPath != "" {
		data, err := os.ReadFile(htmlPath)
		if err != nil {
			return nil, fmt.Errorf("read HTML file: %w", err)
		}
		htmlContent = string(data)
	} else {
		return nil, fmt.Errorf("clipboard or HTML file is required")
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	var KeyMapping = make(map[string]string)

	switch cloudProvider {
	case AWS:
		KeyMapping = map[string]string{
			"Username":          "USERNAME",
			"Password":          "PASSWORD",
			"Access Key Id":     "ACCESS_KEY_ID",
			"Secret Access Key": "SECRET_ACCESS_KEY",
		}
	case Azure:
		KeyMapping = map[string]string{
			"Username":              "USER",
			"Password":              "PASSWORD",
			"Application Client ID": "CLIENT_ID",
			"Application ID":        "CLIENT_ID",
			"Client ID":             "CLIENT_ID",
			"Secret":                "CLIENT_SECRET",
			"Client Secret":         "CLIENT_SECRET",
		}
	}

	creds := make(map[string]string)

	// Extract clientId, clientSecret, username and password
	doc.Find("input").Each(func(i int, s *goquery.Selection) {
		id, exists := s.Attr("id")
		if !exists {
			return
		}

		value, _ := s.Attr("value")
		if mappedKey, ok := KeyMapping[id]; ok {
			creds[mappedKey] = value
		}
	})
	doc.Find("strong").Each(func(i int, s *goquery.Selection) {
		if strings.TrimSpace(s.Text()) == "Sandbox URL" {
			span := s.Parent().Find("span").First()
			url := strings.TrimSpace(span.Text())
			creds["SANDBOX_URL"] = url

			re := regexp.MustCompile(`[?&]region=([^&]+)`)
			matches := re.FindStringSubmatch(url)
			if len(matches) > 1 {
				creds["REGION"] = matches[1]
			}

			if cloudProvider == Azure {
				parts := strings.SplitN(url, "#", 2)
				if len(parts) < 2 {
					return
				}
				fragment := parts[1]
				if strings.HasPrefix(fragment, "@") {
					endIdx := strings.Index(fragment, "/")
					if endIdx > 0 {
						creds["TENANT_NAME"] = fragment[1:endIdx]
					}
				}

				fragParts := strings.Split(fragment, "/")
				for i, p := range fragParts {
					if p == "subscriptions" && i+1 < len(fragParts) {
						creds["SUBSCRIPTION_ID"] = fragParts[i+1]
					}
					if p == "resourceGroups" && i+1 < len(fragParts) {
						creds["RESOURCE_GROUP"] = fragParts[i+1]
					}
				}
			}
		}
	})
	return creds, nil
}

func loadSandboxPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(home, ".config", "bear", "ps", "sandbox_cred.json")
	return path, nil
}

func SaveSandboxCredential(cred SandboxCredential) error {
	path, err := loadSandboxPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cred, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func LoadSandboxCredential() (*PsAzureCredential, error) {
	path, err := loadSandboxPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.New("not logged in: run `bear login`")
	}

	var cred PsAzureCredential
	if err := json.Unmarshal(data, &cred); err != nil {
		return nil, err
	}

	return &cred, nil
}

func LoadAzureSandboxCredential() (*PsAzureCredential, error) {
	return LoadSandboxCredential()
}

func LoadAwsSandboxCredential() (*PsAwsCredential, error) {
	path, err := loadSandboxPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.New("not logged in: run `bear login`")
	}

	var cred PsAwsCredential
	if err := json.Unmarshal(data, &cred); err != nil {
		return nil, err
	}

	return &cred, nil
}

func PurgeSandboxCredential() error {
	path, err := loadSandboxPath()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func RequireSandbox() (*PsAzureCredential, error) {
	return LoadAzureSandboxCredential()
}

func CreatePsAWSCredential(useClipboard bool, filePath string) PsAwsCredential {
	cred, err := CreatePsAWSCredentialWithError(useClipboard, filePath)
	if err != nil {
		log.Fatal(err)
	}
	return cred
}

// CreatePsAWSCredentialWithError creates and stores AWS sandbox credentials
// without exiting the process when validation or persistence fails.
func CreatePsAWSCredentialWithError(useClipboard bool, filePath string) (PsAwsCredential, error) {
	extractorCred, err := ExtractCredential(useClipboard, filePath, AWS)
	if err != nil {
		return PsAwsCredential{}, err
	}

	var psAWSCred PsAwsCredential
	psAWSCred.SandboxURL = extractorCred["SANDBOX_URL"]
	psAWSCred.User = extractorCred["USERNAME"]
	psAWSCred.Password = extractorCred["PASSWORD"]
	psAWSCred.AccessKeyId = extractorCred["ACCESS_KEY_ID"]
	psAWSCred.SecretAccessKey = extractorCred["SECRET_ACCESS_KEY"]
	psAWSCred.Region = extractorCred["REGION"]

	if err := SaveSandboxCredential(&psAWSCred); err != nil {
		return PsAwsCredential{}, fmt.Errorf("save sandbox credential: %w", err)
	}

	return psAWSCred, nil
}

func CreatePsAzureCredential(useClipboard bool, filePath string) (PsAzureCredential, error) {
	extractorCred, err := ExtractCredential(useClipboard, filePath, Azure)
	if err != nil {
		return PsAzureCredential{}, err
	}

	if err := validateAzureExtractorCredential(extractorCred); err != nil {
		return PsAzureCredential{}, err
	}

	var psARMCred PsAzureCredential
	psARMCred.SandboxURL = extractorCred["SANDBOX_URL"]
	psARMCred.SubscriptionID = extractorCred["SUBSCRIPTION_ID"]
	psARMCred.User = extractorCred["USER"]
	psARMCred.Password = extractorCred["PASSWORD"]
	psARMCred.ClientID = extractorCred["CLIENT_ID"]
	psARMCred.ClientSecret = extractorCred["CLIENT_SECRET"]
	psARMCred.ResourceGroup = extractorCred["RESOURCE_GROUP"]
	psARMCred.TenantName = extractorCred["TENANT_NAME"]
	psARMCred.ResourceProviderRegistrations = "none"

	armToken, err := arm.GetARMToken(psARMCred.ClientID, psARMCred.ClientSecret, psARMCred.TenantName)
	if err != nil {
		return PsAzureCredential{}, fmt.Errorf("create azure credential: %w", err)
	}
	psARMCred.TenantID, err = arm.GetTenantID(context.Background(), armToken)
	if err != nil {
		return PsAzureCredential{}, fmt.Errorf("resolve Azure tenant ID: %w", err)
	}

	if err := SaveSandboxCredential(&psARMCred); err != nil {
		return PsAzureCredential{}, fmt.Errorf("failed to save sandbox credential: %w", err)
	}

	return psARMCred, nil
}

func validateAzureExtractorCredential(extractorCred map[string]string) error {
	missing := make([]string, 0, 3)

	if strings.TrimSpace(extractorCred["CLIENT_ID"]) == "" {
		missing = append(missing, "CLIENT_ID")
	}
	if strings.TrimSpace(extractorCred["CLIENT_SECRET"]) == "" {
		missing = append(missing, "CLIENT_SECRET")
	}
	if strings.TrimSpace(extractorCred["TENANT_NAME"]) == "" {
		missing = append(missing, "TENANT_NAME")
	}

	if len(missing) == 0 {
		return nil
	}

	return fmt.Errorf("create azure credential: missing extracted fields %s (ensure clipboard/html contains Azure sandbox credential page and use --cloud-provider aws for AWS labs)", strings.Join(missing, ", "))
}

func DetectOldResourceGroup(content string) (string, bool) {
	var resourceGroupPattern = regexp.MustCompile(
		`\b\d+-[a-z0-9-]+-playground-sandbox\b`,
	)
	match := resourceGroupPattern.FindString(content)
	fmt.Println(content)
	fmt.Println(match)
	return match, match != ""
}

func ReplaceResourceGroupFromSandbox(
	content string,
	sandboxPath string,
) (string, bool, error) {

	cred, err := LoadSandboxCredential()
	if err != nil {
		return "", false, err
	}

	oldRG, found := DetectOldResourceGroup(content)
	if !found {
		return content, false, nil
	}

	updated := strings.ReplaceAll(content, oldRG, cred.ResourceGroup)
	return updated, true, nil
}

func ReplaceResourceGroupInFile(
	filePath string,
	sandboxPath string,
) error {

	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	updated, changed, err := ReplaceResourceGroupFromSandbox(
		string(data),
		sandboxPath,
	)
	if err != nil {
		return err
	}

	if !changed {
		return nil
	}

	return os.WriteFile(filePath, []byte(updated), 0644)
}

func ReplaceResourceGroupInPath(
	path string,
	sandboxPath string,
) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if sandboxPath == "" {
		sandboxPath, _ = loadSandboxPath()
	}
	fmt.Println(path)
	fmt.Println(sandboxPath)

	if info.IsDir() {
		return filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			// Optional: filter file types
			switch filepath.Ext(p) {
			case ".tf", ".tfvars", ".txt", ".sh":
				return ReplaceResourceGroupInFile(p, sandboxPath)
			default:
				return nil
			}
		})
	}

	return ReplaceResourceGroupInFile(path, sandboxPath)
}

func LoginAzurePortalFromSandbox() error {
	cred, err := LoadSandboxCredential()
	if err != nil {
		return err
	}

	if cred.User == "" || cred.Password == "" {
		return fmt.Errorf("sandbox credential missing username or password")
	}

	if cred.SandboxURL == "" {
		return fmt.Errorf("sandbox credential missing portal URL")
	}

	browser.LoginInBrowser(cred.User, cred.Password, browser.AzurePortal, string(browser.AzurePortal))
	return nil
}

func SetupTerraform(cloudProvider CloudProvider) (*TerraformSetup, error) {
	switch cloudProvider {
	case AWS:
		cred, err := LoadAwsSandboxCredential()
		if err != nil {
			return nil, err
		}

		return &TerraformSetup{
			BackendType:  "s3",
			ResourceType: "s3 bucket",
			Environment:  cred.ToTerraformEnvMap(),
		}, nil
	case Azure:
		cred, err := LoadAzureSandboxCredential()
		if err != nil {
			return nil, err
		}

		return &TerraformSetup{
			BackendType:  "azurerm",
			ResourceType: "storage account",
			Environment:  cred.ToTerraformEnvMap(),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported cloud provider: %s", cloudProvider)
	}
}

func RemoveTerraformStateFiles(root string) error {
	stateFiles := map[string]struct{}{
		"terraform.tfstate":        {},
		"terraform.tfstate.backup": {},
	}

	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Match terraform state files
		if _, ok := stateFiles[d.Name()]; ok {
			if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("failed to remove %s: %w", path, err)
			}
		}

		return nil
	})
}

func SetUpTerraformBackend(clientId, clientSecret, tenant, subscriptionID, resourceGroup, accountName, location, containerName string) (*TerraformSetup, error) {
	if strings.TrimSpace(clientId) == "" || strings.TrimSpace(clientSecret) == "" || strings.TrimSpace(tenant) == "" {
		return nil, fmt.Errorf("client ID, client secret, and tenant are required")
	}
	if strings.TrimSpace(subscriptionID) == "" || strings.TrimSpace(resourceGroup) == "" {
		return nil, fmt.Errorf("subscription ID and resource group are required")
	}
	if strings.TrimSpace(accountName) == "" {
		return nil, fmt.Errorf("storage account name is required")
	}
	if strings.TrimSpace(location) == "" {
		return nil, fmt.Errorf("location is required")
	}
	if strings.TrimSpace(containerName) == "" {
		return nil, fmt.Errorf("container name is required")
	}

	token, err := arm.GetARMToken(clientId, clientSecret, tenant)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("failed to acquire ARM token")
	}

	saParams := arm.CreateStorageAccountParams{
		SubscriptionID: subscriptionID,
		ResourceGroup:  resourceGroup,
		AccountName:    accountName,
		Location:       location,
	}

	if err := arm.CreateStorageAccount(token, saParams); err != nil {
		return nil, err
	}

	containerParams := arm.CreateStorageContainerParams{
		SubscriptionID: subscriptionID,
		ResourceGroup:  resourceGroup,
		AccountName:    accountName,
		ContainerName:  containerName,
	}

	if err := arm.CreateStorageContainer(token, containerParams); err != nil {
		return nil, err
	}

	connectionString, err := arm.GetStorageAccountConnectionString(token, containerParams)
	if err != nil {
		return nil, err
	}

	tenantID, err := arm.GetTenantID(context.Background(), token)
	if err != nil {
		return nil, err
	}

	return &TerraformSetup{
		BackendType:  "azurerm",
		ResourceType: "storage account and blob container",
		Environment: map[string]string{
			"ARM_SUBSCRIPTION_ID":                 subscriptionID,
			"ARM_TENANT_ID":                       tenantID,
			"ARM_CLIENT_ID":                       clientId,
			"ARM_CLIENT_SECRET":                   clientSecret,
			"ARM_RESOURCE_PROVIDER_REGISTRATIONS": "none",
			"ARM_LOCATION":                        location,
			"TF_BACKEND_RESOURCE_GROUP_NAME":      resourceGroup,
			"TF_BACKEND_STORAGE_ACCOUNT_NAME":     accountName,
			"TF_BACKEND_CONTAINER_NAME":           containerName,
			"TF_BACKEND_KEY":                      "terraform.tfstate",
			"TF_BACKEND_CONNECTION_STRING":        connectionString,
			"AZURE_STORAGE_CONNECTION_STRING":     connectionString,
		},
	}, nil
}
