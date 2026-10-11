package aws

import (
	codecommitsvc "bear_cli/internal/aws"
	"bear_cli/pkg/prompt"
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var CodeCommitCmd = &cobra.Command{
	Use:   "aws",
	Short: "AWS CodeCommit utilities",
	Long:  "Create repositories and upload local project files to AWS CodeCommit.",
}

func init() {
	CodeCommitCmd.AddCommand(createRepositoryCmd())
	CodeCommitCmd.AddCommand(pushProjectCmd())
}

type createRepositoryOptions struct {
	RepoName    string
	Description string
	Region      string
	Profile     string
}

func createRepositoryCmd() *cobra.Command {
	opts := &createRepositoryOptions{}

	cmd := &cobra.Command{
		Use:   "create-repo",
		Short: "Create a CodeCommit repository with the AWS SDK",
		RunE: func(cmd *cobra.Command, args []string) error {
			stopSpinner := prompt.StartSpinner("Creating CodeCommit repository...")
			result, err := codecommitsvc.CreateRepository(context.Background(), codecommitsvc.CreateRepositoryInput{
				Repository:  opts.RepoName,
				Description: opts.Description,
				Region:      opts.Region,
				Profile:     opts.Profile,
			})
			stopSpinner(err)
			if err != nil {
				return err
			}

			status := "already exists"
			if result.RepositoryCreated {
				status = "created"
			}

			fmt.Printf("Repository %s: %s\n", status, result.Repository)
			if result.CloneURLHTTP != "" {
				fmt.Printf("HTTPS clone URL: %s\n", result.CloneURLHTTP)
			}
			if result.CloneURLSSH != "" {
				fmt.Printf("SSH clone URL: %s\n", result.CloneURLSSH)
			}
			if result.DefaultBranch != "" {
				fmt.Printf("Default branch: %s\n", result.DefaultBranch)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&opts.RepoName, "repo", "", "CodeCommit repository name (required)")
	cmd.Flags().StringVar(&opts.Description, "description", "", "Repository description")
	cmd.Flags().StringVar(&opts.Region, "region", "", "AWS region override")
	cmd.Flags().StringVar(&opts.Profile, "profile", "", "AWS shared profile override")

	cmd.MarkFlagRequired("repo")

	return cmd
}

type pushProjectOptions struct {
	Path        string
	RepoName    string
	Branch      string
	Description string
	Region      string
	Profile     string
	AuthorName  string
	AuthorEmail string
}

func pushProjectCmd() *cobra.Command {
	defaultAuthor := os.Getenv("USER")
	if defaultAuthor == "" {
		defaultAuthor = "bear-cli"
	}

	opts := &pushProjectOptions{}

	cmd := &cobra.Command{
		Use:   "push-project",
		Short: "Create or update a CodeCommit repository from a local project folder",
		RunE: func(cmd *cobra.Command, args []string) error {
			stopSpinner := prompt.StartSpinner("Pushing project to CodeCommit...")
			result, err := codecommitsvc.PushProject(context.Background(), codecommitsvc.PushProjectInput{
				Path:        opts.Path,
				Repository:  opts.RepoName,
				Branch:      opts.Branch,
				Description: opts.Description,
				Region:      opts.Region,
				Profile:     opts.Profile,
				AuthorName:  opts.AuthorName,
				AuthorEmail: opts.AuthorEmail,
			})
			stopSpinner(err)
			if err != nil {
				return err
			}

			status := "updated"
			if result.RepositoryCreated {
				status = "created"
			}

			fmt.Printf("Repository %s: %s\n", status, result.Repository)
			if result.CloneURLHTTP != "" {
				fmt.Printf("Clone URL: %s\n", result.CloneURLHTTP)
			}
			fmt.Printf("Branch: %s\n", result.Branch)
			fmt.Printf("Files uploaded: %d\n", result.FilesUploaded)
			fmt.Printf("Commits created: %d\n", len(result.CommitIDs))
			if len(result.CommitIDs) > 0 {
				fmt.Printf("Latest commit: %s\n", result.CommitIDs[len(result.CommitIDs)-1])
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Path, "path", "", "Local project path or public GitHub URL to upload (required)")
	cmd.Flags().StringVar(&opts.RepoName, "repo", "", "CodeCommit repository name (required)")
	cmd.Flags().StringVar(&opts.Branch, "branch", "main", "Target branch name")
	cmd.Flags().StringVar(&opts.Description, "description", "", "Repository description")
	cmd.Flags().StringVar(&opts.Region, "region", "", "AWS region override")
	cmd.Flags().StringVar(&opts.Profile, "profile", "", "AWS shared profile override")
	cmd.Flags().StringVar(&opts.AuthorName, "author-name", defaultAuthor, "Commit author name")
	cmd.Flags().StringVar(&opts.AuthorEmail, "author-email", "", "Commit author email")

	cmd.MarkFlagRequired("path")
	cmd.MarkFlagRequired("repo")

	return cmd
}
