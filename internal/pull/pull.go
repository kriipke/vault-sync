package pull

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"vault-sync/internal/config"
	"vault-sync/internal/vault"
)

type Puller struct {
	client *vault.Client
	config *config.Config
}

func New(client *vault.Client, cfg *config.Config) *Puller {
	return &Puller{
		client: client,
		config: cfg,
	}
}

func (p *Puller) Pull(ctx context.Context) error {
	fmt.Printf("Pulling secrets from Vault to %s\n", p.config.OutputDir)
	
	if err := os.MkdirAll(p.config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	secretCount := 0
	err := p.client.WalkSecrets(ctx, p.config.BasePath, func(secretPath string) error {
		if err := p.pullSecret(ctx, secretPath); err != nil {
			return fmt.Errorf("failed to pull secret %s: %w", secretPath, err)
		}
		secretCount++
		fmt.Printf("✓ Pulled: %s\n", secretPath)
		return nil
	})

	if err != nil {
		return err
	}

	fmt.Printf("\nSuccessfully pulled %d secrets\n", secretCount)
	return nil
}

func (p *Puller) pullSecret(ctx context.Context, secretPath string) error {
	secret, err := p.client.ReadSecret(ctx, secretPath)
	if err != nil {
		return err
	}

	localPath := p.getLocalPath(secretPath)
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", localPath, err)
	}

	yamlData, err := yaml.Marshal(secret.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal secret data: %w", err)
	}

	if err := os.WriteFile(localPath, yamlData, 0600); err != nil {
		return fmt.Errorf("failed to write file %s: %w", localPath, err)
	}

	return nil
}

func (p *Puller) getLocalPath(secretPath string) string {
	cleanPath := strings.TrimPrefix(secretPath, "/")
	if p.config.BasePath != "" {
		cleanPath = strings.TrimPrefix(cleanPath, strings.TrimPrefix(p.config.BasePath, "/"))
		cleanPath = strings.TrimPrefix(cleanPath, "/")
	}
	
	return filepath.Join(p.config.OutputDir, cleanPath+".yaml")
}