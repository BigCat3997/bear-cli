package ps

import (
	"bear_cli/internal/armapi"
	"bear_cli/internal/browser"
	"bear_cli/models"
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

func RunExtractor(useClipboard bool, htmlPath string, cloudProvider models.CloudProvider) map[string]string {
	var htmlContent string
	var err error

	if useClipboard {
		htmlContent, err = clipboard.ReadAll()
		if err != nil {
			log.Fatal("Failed to read clipboard:", err)
		}
	} else if htmlPath != "" {
		data, err := os.ReadFile(htmlPath)
		if err != nil {
			log.Fatal("Failed to read file:", err)
		}
		htmlContent = string(data)
	} else {
		log.Fatal("Either --file or --clipboard must be provided")
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		log.Fatal("Failed to parse HTML:", err)
	}

	var KeyMapping = make(map[string]string)

	switch cloudProvider {
	case models.AWS:
		KeyMapping = map[string]string{
			"Username":          "USERNAME",
			"Password":          "PASSWORD",
			"Access Key Id":     "ACCESS_KEY_ID",
			"Secret Access Key": "SECRET_ACCESS_KEY",
		}
	case models.Azure:
		KeyMapping = map[string]string{
			"Username":              "USER",
			"Password":              "PASSWORD",
			"Application Client ID": "CLIENT_ID",
			"Secret":                "CLIENT_SECRET",
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

			if cloudProvider == models.Azure {
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
	return creds
}

func loadSandboxPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(home, ".config", "bear", "ps", "sandbox_cred.json")
	return path, nil
}

func SaveSandboxCredential(cred models.SandboxCredential) error {
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

func LoadSandboxCredential() (*models.PsAzureCredential, error) {
	path, err := loadSandboxPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.New("not logged in: run `bear login`")
	}

	var cred models.PsAzureCredential
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

func RequireSandbox() (*models.PsAzureCredential, error) {
	return LoadSandboxCredential()
}

func CreatePsAWSCredential(useClipboard bool, filePath string) models.PsAwsCredential {
	extractorCred := RunExtractor(useClipboard, filePath, models.AWS)

	var psAWSCred models.PsAwsCredential
	psAWSCred.SandboxURL = extractorCred["SANDBOX_URL"]
	psAWSCred.User = extractorCred["USERNAME"]
	psAWSCred.Password = extractorCred["PASSWORD"]
	psAWSCred.AccessKeyId = extractorCred["ACCESS_KEY_ID"]
	psAWSCred.SecretAccessKey = extractorCred["SECRET_ACCESS_KEY"]
	psAWSCred.Region = extractorCred["REGION"]

	if err := SaveSandboxCredential(&psAWSCred); err != nil {
		log.Fatalf("failed to save sandbox credential: %v", err)
	}

	return psAWSCred
}

func CreatePsAzureCredential(useClipboard bool, filePath string) models.PsAzureCredential {
	extractorCred := RunExtractor(useClipboard, filePath, models.Azure)

	var psARMCred models.PsAzureCredential
	psARMCred.SandboxURL = extractorCred["SANDBOX_URL"]
	psARMCred.SubscriptionID = extractorCred["SUBSCRIPTION_ID"]
	psARMCred.User = extractorCred["USER"]
	psARMCred.Password = extractorCred["PASSWORD"]
	psARMCred.ClientID = extractorCred["CLIENT_ID"]
	psARMCred.ClientSecret = extractorCred["CLIENT_SECRET"]
	psARMCred.ResourceGroup = extractorCred["RESOURCE_GROUP"]
	psARMCred.TenantName = extractorCred["TENANT_NAME"]
	psARMCred.ResourceProviderRegistrations = "none"

	armToken := armapi.GetARMToken(psARMCred.ClientID, psARMCred.ClientSecret, psARMCred.TenantName)
	psARMCred.TenantID = armapi.GetTenantId(armToken)

	if err := SaveSandboxCredential(&psARMCred); err != nil {
		log.Fatalf("failed to save sandbox credential: %v", err)
	}

	return psARMCred
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
