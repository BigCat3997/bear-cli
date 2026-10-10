package terraform

import (
	"bear_cli/internal/config"
	"bear_cli/pkg/prompt"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type cleanOptions struct {
	Path      string
	Recursive bool
}

// cleanCmd removes ".terraform" directories and Terraform state files
// (terraform.tfstate, terraform.tfstate.backup, and .terraform.lock.hcl) from
// the configured Terraform working directory.
func cleanCmd(cfg config.TerraformConfig) *cobra.Command {
	opts := &cleanOptions{Path: cfg.Path}

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Remove .terraform directories and Terraform state files",
		Long:  "Remove .terraform directories and Terraform state files from the working directory. Use --recursive to also clean matching subdirectories.",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := strings.TrimSpace(opts.Path)
			if path == "" {
				return fmt.Errorf("terraform path is not set; pass --path or set [terraform].path in config.toml")
			}

			info, err := os.Stat(path)
			if err != nil {
				return fmt.Errorf("resolve path %q: %w", path, err)
			}
			if !info.IsDir() {
				return fmt.Errorf("path %q is not a directory", path)
			}

			stopSpinner := prompt.StartSpinner(fmt.Sprintf("Cleaning Terraform artifacts in %s...", path))
			removed, err := clean(path, opts.Recursive)
			stopSpinner(err)
			if err != nil {
				return err
			}

			if len(removed) == 0 {
				fmt.Println("nothing to clean")
				return nil
			}

			for _, entry := range removed {
				fmt.Printf("removed %s\n", entry)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Path, "path", opts.Path, "Terraform working directory to clean (default: [terraform].path in config.toml)")
	cmd.Flags().BoolVarP(&opts.Recursive, "recursive", "r", false, "Also clean matching entries in subdirectories")

	return cmd
}

func isTerraformStateFile(name string) bool {
	switch name {
	case "terraform.tfstate", "terraform.tfstate.backup", ".terraform.lock.hcl":
		return true
	}
	return strings.HasPrefix(name, "terraform.tfstate.") && strings.HasSuffix(name, ".backup")
}

// clean removes .terraform directories and Terraform state files under root.
// When recursive is false, only direct children of root are considered.
func clean(root string, recursive bool) ([]string, error) {
	var removed []string

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read directory %q: %w", root, err)
	}

	for _, entry := range entries {
		fullPath := filepath.Join(root, entry.Name())

		if entry.IsDir() {
			if entry.Name() == ".terraform" {
				if err := os.RemoveAll(fullPath); err != nil {
					return removed, fmt.Errorf("remove %q: %w", fullPath, err)
				}
				removed = append(removed, fullPath)
				continue
			}

			if recursive {
				nested, err := clean(fullPath, recursive)
				removed = append(removed, nested...)
				if err != nil {
					return removed, err
				}
			}
			continue
		}

		if isTerraformStateFile(entry.Name()) {
			if err := os.Remove(fullPath); err != nil {
				return removed, fmt.Errorf("remove %q: %w", fullPath, err)
			}
			removed = append(removed, fullPath)
		}
	}

	return removed, nil
}
