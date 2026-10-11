package aws

import (
	"archive/zip"
	ps "bear_cli/internal/pluralsight"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awscodecommit "github.com/aws/aws-sdk-go-v2/service/codecommit"
	"github.com/aws/aws-sdk-go-v2/service/codecommit/types"
	"github.com/aws/smithy-go"
	ignore "github.com/sabhiram/go-gitignore"
)

const maxFilesPerCommit = 100

type CreateRepositoryInput struct {
	Repository  string
	Description string
	Region      string
	Profile     string
}

type CreateRepositoryResult struct {
	Repository        string
	CloneURLHTTP      string
	CloneURLSSH       string
	DefaultBranch     string
	RepositoryCreated bool
}

type PushProjectInput struct {
	Path        string
	Repository  string
	Branch      string
	Description string
	Region      string
	Profile     string
	AuthorName  string
	AuthorEmail string
}

type PushProjectResult struct {
	Repository        string
	CloneURLHTTP      string
	Branch            string
	FilesUploaded     int
	CommitIDs         []string
	RepositoryCreated bool
}

type projectFile struct {
	Path    string
	Content []byte
	Mode    types.FileModeTypeEnum
}

func CreateRepository(ctx context.Context, input CreateRepositoryInput) (*CreateRepositoryResult, error) {
	if strings.TrimSpace(input.Repository) == "" {
		return nil, errors.New("repository name is required")
	}
	if strings.HasSuffix(input.Repository, ".git") {
		return nil, errors.New("repository name cannot end with .git")
	}

	client, err := newClient(ctx, input.Region, input.Profile)
	if err != nil {
		return nil, err
	}

	repoMetadata, created, err := ensureRepository(ctx, client, input.Repository, input.Description)
	if err != nil {
		return nil, err
	}

	result := &CreateRepositoryResult{
		Repository:        input.Repository,
		RepositoryCreated: created,
	}
	if repoMetadata != nil {
		result.CloneURLHTTP = aws.ToString(repoMetadata.CloneUrlHttp)
		result.CloneURLSSH = aws.ToString(repoMetadata.CloneUrlSsh)
		result.DefaultBranch = aws.ToString(repoMetadata.DefaultBranch)
	}

	return result, nil
}

func PushProject(ctx context.Context, input PushProjectInput) (*PushProjectResult, error) {
	if strings.TrimSpace(input.Path) == "" {
		return nil, errors.New("path or public GitHub URL is required")
	}
	if strings.TrimSpace(input.Repository) == "" {
		return nil, errors.New("repository name is required")
	}
	if strings.HasSuffix(input.Repository, ".git") {
		return nil, errors.New("repository name cannot end with .git")
	}
	if strings.TrimSpace(input.Branch) == "" {
		input.Branch = "main"
	}

	resolvedPath, cleanup, err := resolveSourcePath(input.Path)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	info, err := os.Stat(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("stat project path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path must be a directory: %s", resolvedPath)
	}

	files, err := collectProjectFiles(resolvedPath)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no files found in %s", resolvedPath)
	}

	client, err := newClient(ctx, input.Region, input.Profile)
	if err != nil {
		return nil, err
	}
	repoMetadata, created, err := ensureRepository(ctx, client, input.Repository, input.Description)
	if err != nil {
		return nil, err
	}

	parentCommitID, err := resolveBranchHead(ctx, client, input.Repository, input.Branch)
	if err != nil {
		return nil, err
	}

	result := &PushProjectResult{
		Repository:        input.Repository,
		Branch:            input.Branch,
		FilesUploaded:     len(files),
		RepositoryCreated: created,
	}
	if repoMetadata != nil && repoMetadata.CloneUrlHttp != nil {
		result.CloneURLHTTP = aws.ToString(repoMetadata.CloneUrlHttp)
	}

	for start := 0; start < len(files); start += maxFilesPerCommit {
		end := start + maxFilesPerCommit
		if end > len(files) {
			end = len(files)
		}

		commitID, err := createCommit(ctx, client, input, files[start:end], parentCommitID, start == 0, len(files))
		if err != nil {
			return nil, err
		}

		result.CommitIDs = append(result.CommitIDs, commitID)
		parentCommitID = aws.String(commitID)
	}

	return result, nil
}

func resolveSourcePath(pathOrURL string) (string, func(), error) {
	trimmed := strings.TrimSpace(pathOrURL)
	if trimmed == "" {
		return "", func() {}, errors.New("path or public GitHub URL is required")
	}

	if isGitHubHTTPURL(trimmed) {
		return downloadGitHubRepoSnapshot(trimmed)
	}

	absPath, err := filepath.Abs(trimmed)
	if err != nil {
		return "", func() {}, fmt.Errorf("resolve project path: %w", err)
	}

	return absPath, func() {}, nil
}

func isGitHubHTTPURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	host := strings.ToLower(u.Host)
	return host == "github.com" || host == "www.github.com"
}

func downloadGitHubRepoSnapshot(repoURL string) (string, func(), error) {
	owner, repo, err := parseGitHubOwnerRepo(repoURL)
	if err != nil {
		return "", func() {}, err
	}

	tmpDir, err := os.MkdirTemp("", "bear-github-source-*")
	if err != nil {
		return "", func() {}, fmt.Errorf("create temp directory: %w", err)
	}

	cleanup := func() {
		_ = os.RemoveAll(tmpDir)
	}

	branch, err := fetchGitHubDefaultBranch(owner, repo)
	if err != nil {
		cleanup()
		return "", func() {}, err
	}

	zipPath := filepath.Join(tmpDir, "repo.zip")
	archiveURL := fmt.Sprintf("https://codeload.github.com/%s/%s/zip/refs/heads/%s", owner, repo, branch)
	if err := downloadToFile(archiveURL, zipPath); err != nil {
		cleanup()
		return "", func() {}, err
	}

	extractDir := filepath.Join(tmpDir, "extract")
	if err := unzipArchive(zipPath, extractDir); err != nil {
		cleanup()
		return "", func() {}, err
	}

	entries, err := os.ReadDir(extractDir)
	if err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("read extracted directory: %w", err)
	}
	if len(entries) == 0 {
		cleanup()
		return "", func() {}, errors.New("downloaded GitHub archive is empty")
	}

	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(extractDir, entry.Name()), cleanup, nil
		}
	}

	return extractDir, cleanup, nil
}

func parseGitHubOwnerRepo(raw string) (string, string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", fmt.Errorf("parse GitHub URL: %w", err)
	}

	parts := strings.Split(strings.Trim(strings.TrimSpace(u.Path), "/"), "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid GitHub repository URL: %s", raw)
	}

	owner := parts[0]
	repo := strings.TrimSuffix(parts[1], ".git")
	if owner == "" || repo == "" {
		return "", "", fmt.Errorf("invalid GitHub repository URL: %s", raw)
	}

	return owner, repo, nil
}

func fetchGitHubDefaultBranch(owner, repo string) (string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("build GitHub API request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "bear-cli")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request GitHub repository metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("unable to access public GitHub repo (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var metadata struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return "", fmt.Errorf("decode GitHub repository metadata: %w", err)
	}
	if strings.TrimSpace(metadata.DefaultBranch) == "" {
		return "", errors.New("GitHub repository default branch is empty")
	}

	return metadata.DefaultBranch, nil
}

func downloadToFile(rawURL, path string) error {
	resp, err := http.Get(rawURL)
	if err != nil {
		return fmt.Errorf("download repository archive: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("download repository archive failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create archive file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("write archive file: %w", err)
	}

	return nil
}

func unzipArchive(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip archive: %w", err)
	}
	defer r.Close()

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("create extract directory: %w", err)
	}

	for _, f := range r.File {
		targetPath := filepath.Join(destDir, f.Name)
		cleanTargetPath := filepath.Clean(targetPath)
		cleanDestDir := filepath.Clean(destDir) + string(os.PathSeparator)
		if !strings.HasPrefix(cleanTargetPath, cleanDestDir) {
			return fmt.Errorf("invalid archive entry: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(cleanTargetPath, f.Mode()); err != nil {
				return fmt.Errorf("create directory %s: %w", cleanTargetPath, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(cleanTargetPath), 0755); err != nil {
			return fmt.Errorf("create parent directory %s: %w", cleanTargetPath, err)
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open archive file %s: %w", f.Name, err)
		}

		out, err := os.OpenFile(cleanTargetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return fmt.Errorf("create extracted file %s: %w", cleanTargetPath, err)
		}

		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			rc.Close()
			return fmt.Errorf("extract file %s: %w", cleanTargetPath, err)
		}

		if err := out.Close(); err != nil {
			rc.Close()
			return fmt.Errorf("close extracted file %s: %w", cleanTargetPath, err)
		}
		if err := rc.Close(); err != nil {
			return fmt.Errorf("close archive file %s: %w", f.Name, err)
		}
	}

	return nil
}

func collectProjectFiles(rootPath string) ([]projectFile, error) {
	files := make([]projectFile, 0)
	ignorer, err := loadProjectIgnore(rootPath)
	if err != nil {
		return nil, err
	}

	walkErr := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}
		relPath = filepath.ToSlash(relPath)

		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			if shouldIgnoreProjectPath(ignorer, relPath, true) {
				return filepath.SkipDir
			}
			return nil
		}

		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink is not supported: %s", path)
		}
		if shouldIgnoreProjectPath(ignorer, relPath, false) {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read file %s: %w", path, err)
		}

		mode := types.FileModeTypeEnumNormal
		if isExecutable(d) {
			mode = types.FileModeTypeEnumExecutable
		}

		files = append(files, projectFile{
			Path:    filepath.ToSlash(relPath),
			Content: content,
			Mode:    mode,
		})

		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("walk project path: %w", walkErr)
	}

	return files, nil
}

func loadProjectIgnore(rootPath string) (*ignore.GitIgnore, error) {
	ignoreFilePath := filepath.Join(rootPath, ".gitignore")
	if _, err := os.Stat(ignoreFilePath); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat .gitignore: %w", err)
	}

	ignorer, err := ignore.CompileIgnoreFile(ignoreFilePath)
	if err != nil {
		return nil, fmt.Errorf("compile .gitignore: %w", err)
	}

	return ignorer, nil
}

func shouldIgnoreProjectPath(ignorer *ignore.GitIgnore, relPath string, isDir bool) bool {
	if ignorer == nil {
		return false
	}

	if ignorer.MatchesPath(relPath) {
		return true
	}

	return isDir && ignorer.MatchesPath(relPath+"/")
}

func isExecutable(entry fs.DirEntry) bool {
	info, err := entry.Info()
	if err != nil {
		return false
	}

	return info.Mode().Perm()&0111 != 0
}

func ensureRepository(ctx context.Context, client *awscodecommit.Client, name, description string) (*types.RepositoryMetadata, bool, error) {
	getOutput, err := client.GetRepository(ctx, &awscodecommit.GetRepositoryInput{
		RepositoryName: aws.String(name),
	})
	if err == nil {
		return getOutput.RepositoryMetadata, false, nil
	}
	if !hasErrorCode(err, "RepositoryDoesNotExistException") {
		return nil, false, fmt.Errorf("get repository: %w", err)
	}

	createOutput, err := client.CreateRepository(ctx, &awscodecommit.CreateRepositoryInput{
		RepositoryName:        aws.String(name),
		RepositoryDescription: aws.String(description),
	})
	if err != nil {
		return nil, false, fmt.Errorf("create repository: %w", err)
	}

	return createOutput.RepositoryMetadata, true, nil
}

func resolveBranchHead(ctx context.Context, client *awscodecommit.Client, repositoryName, branchName string) (*string, error) {
	output, err := client.GetBranch(ctx, &awscodecommit.GetBranchInput{
		RepositoryName: aws.String(repositoryName),
		BranchName:     aws.String(branchName),
	})
	if err != nil {
		if hasErrorCode(err, "BranchDoesNotExistException") || hasErrorCode(err, "CommitDoesNotExistException") {
			return nil, nil
		}
		return nil, fmt.Errorf("get branch %s: %w", branchName, err)
	}
	if output.Branch == nil || output.Branch.CommitId == nil {
		return nil, nil
	}

	return output.Branch.CommitId, nil
}

func createCommit(ctx context.Context, client *awscodecommit.Client, input PushProjectInput, files []projectFile, parentCommitID *string, firstChunk bool, totalFiles int) (string, error) {
	putFiles := make([]types.PutFileEntry, 0, len(files))
	for _, file := range files {
		putFiles = append(putFiles, types.PutFileEntry{
			FilePath:    aws.String(file.Path),
			FileContent: file.Content,
			FileMode:    file.Mode,
		})
	}

	message := fmt.Sprintf("Upload %d files from %s", totalFiles, filepath.Base(input.Path))
	if !firstChunk {
		message = fmt.Sprintf("Continue upload for %s", filepath.Base(input.Path))
	}

	request := &awscodecommit.CreateCommitInput{
		BranchName:     aws.String(input.Branch),
		RepositoryName: aws.String(input.Repository),
		AuthorName:     nilIfBlank(input.AuthorName),
		Email:          nilIfBlank(input.AuthorEmail),
		CommitMessage:  aws.String(message),
		PutFiles:       putFiles,
	}
	if parentCommitID != nil && strings.TrimSpace(aws.ToString(parentCommitID)) != "" {
		request.ParentCommitId = parentCommitID
	}

	output, err := client.CreateCommit(ctx, request)
	if err != nil {
		return "", fmt.Errorf("create commit: %w", err)
	}
	if output.CommitId == nil || aws.ToString(output.CommitId) == "" {
		return "", errors.New("create commit did not return a commit ID")
	}

	return aws.ToString(output.CommitId), nil
}

func nilIfBlank(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}

	return aws.String(trimmed)
}

func hasErrorCode(err error, code string) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode() == code
	}

	return false
}

func newClient(ctx context.Context, region, profile string) (*awscodecommit.Client, error) {
	// Try to load credentials from PluralSight sandbox JSON file first
	loadOptions := []func(*config.LoadOptions) error{}

	sandboxCred, err := ps.LoadAwsSandboxCredential()
	if err == nil && sandboxCred != nil {
		// Use PluralSight credentials if available
		if strings.TrimSpace(sandboxCred.AccessKeyId) != "" && strings.TrimSpace(sandboxCred.SecretAccessKey) != "" {
			staticCreds := credentials.NewStaticCredentialsProvider(
				sandboxCred.AccessKeyId,
				sandboxCred.SecretAccessKey,
				"", // no session token
			)
			loadOptions = append(loadOptions, config.WithCredentialsProvider(staticCreds))

			// Use region from sandbox credentials if no explicit region provided
			if strings.TrimSpace(region) == "" && strings.TrimSpace(sandboxCred.Region) != "" {
				region = sandboxCred.Region
			}
		}
	}

	// Add region if provided or fall back to discovered region
	if strings.TrimSpace(region) != "" {
		loadOptions = append(loadOptions, config.WithRegion(region))
	}

	// Add profile if provided
	if strings.TrimSpace(profile) != "" {
		loadOptions = append(loadOptions, config.WithSharedConfigProfile(profile))
	}

	cfg, err := config.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}
	if strings.TrimSpace(cfg.Region) == "" {
		return nil, errors.New("AWS region is required; set --region, AWS_REGION, or a profile region")
	}

	return awscodecommit.NewFromConfig(cfg), nil
}
