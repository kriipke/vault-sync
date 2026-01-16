package push

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"vault-sync/internal/config"
	"vault-sync/internal/diff"
	"vault-sync/internal/vault"
)

type Pusher struct {
	client *vault.Client
	config *config.Config
}

func New(client *vault.Client, cfg *config.Config) *Pusher {
	return &Pusher{
		client: client,
		config: cfg,
	}
}

func (p *Pusher) Push(ctx context.Context) error {
	fmt.Printf("Pushing secrets from %s to Vault\n", p.config.OutputDir)
	
	if _, err := os.Stat(p.config.OutputDir); os.IsNotExist(err) {
		return fmt.Errorf("output directory %s does not exist", p.config.OutputDir)
	}

	var localSecrets []*vault.Secret
	err := filepath.Walk(p.config.OutputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".yaml") {
			secret, err := p.loadLocalSecret(path)
			if err != nil {
				return fmt.Errorf("failed to load %s: %w", path, err)
			}
			localSecrets = append(localSecrets, secret)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk output directory: %w", err)
	}

	pushCount := 0
	for _, localSecret := range localSecrets {
		shouldPush, err := p.processSecret(ctx, localSecret)
		if err != nil {
			return fmt.Errorf("failed to process secret %s: %w", localSecret.Path, err)
		}

		if shouldPush {
			pushCount++
		}
	}

	fmt.Printf("\nProcessed %d local secrets, pushed %d changes\n", len(localSecrets), pushCount)
	return nil
}

func (p *Pusher) processSecret(ctx context.Context, localSecret *vault.Secret) (bool, error) {
	fmt.Printf("\nProcessing: %s\n", localSecret.Path)

	currentSecret, err := p.client.ReadSecret(ctx, localSecret.Path)
	if err != nil {
		fmt.Printf("Secret %s does not exist in Vault (will create new)\n", localSecret.Path)
		currentSecret = &vault.Secret{
			Path: localSecret.Path,
			Data: make(map[string]string),
		}
	}

	secretDiff, err := diff.CompareSecrets(currentSecret, localSecret)
	if err != nil {
		return false, fmt.Errorf("failed to compare secrets: %w", err)
	}

	if !secretDiff.HasDiff {
		fmt.Printf("✓ No changes needed for %s\n", localSecret.Path)
		return false, nil
	}

	fmt.Println("Changes detected:")
	diff.PrintDiff(secretDiff)

	if p.config.DryRun {
		fmt.Printf("✓ [DRY RUN] Would update %s\n", localSecret.Path)
		return false, nil
	}

	if !p.config.AutoApprove {
		if !p.promptForApproval(localSecret.Path) {
			fmt.Printf("✗ Skipped %s\n", localSecret.Path)
			return false, nil
		}
	}

	if err := p.client.WriteSecret(ctx, localSecret); err != nil {
		return false, fmt.Errorf("failed to write secret: %w", err)
	}

	fmt.Printf("✓ Updated %s\n", localSecret.Path)
	return true, nil
}

func (p *Pusher) loadLocalSecret(filePath string) (*vault.Secret, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var secretData map[string]string
	if err := yaml.Unmarshal(data, &secretData); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	vaultPath := p.getVaultPath(filePath)
	
	return &vault.Secret{
		Path: vaultPath,
		Data: secretData,
	}, nil
}

func (p *Pusher) getVaultPath(filePath string) string {
	relPath, err := filepath.Rel(p.config.OutputDir, filePath)
	if err != nil {
		relPath = filePath
	}

	vaultPath := strings.TrimSuffix(relPath, ".yaml")
	vaultPath = strings.ReplaceAll(vaultPath, string(filepath.Separator), "/")
	
	if p.config.BasePath != "" {
		basePath := strings.TrimPrefix(p.config.BasePath, "/")
		basePath = strings.TrimSuffix(basePath, "/")
		if basePath != "" {
			vaultPath = basePath + "/" + vaultPath
		}
	}

	return vaultPath
}

func (p *Pusher) promptForApproval(secretPath string) bool {
	fmt.Printf("Apply changes to %s? [y/N]: ", secretPath)
	
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}