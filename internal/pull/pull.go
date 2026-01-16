package pull

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"vault-sync/internal/config"
	"vault-sync/internal/errors"
	"vault-sync/internal/logger"
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
	start := time.Now()
	logger.InfoCtx(ctx, "Starting pull operation", 
		"output_dir", p.config.OutputDir,
		"base_path", p.config.BasePath)
	
	fmt.Printf("Pulling secrets from Vault to %s\n", p.config.OutputDir)
	
	if err := os.MkdirAll(p.config.OutputDir, 0755); err != nil {
		return errors.New("create_output_dir", err).
			WithContext("output_dir", p.config.OutputDir)
	}

	logger.DebugCtx(ctx, "Created output directory", "path", p.config.OutputDir)

	secretCount := 0
	err := p.client.WalkSecrets(ctx, p.config.BasePath, func(secretPath string) error {
		if err := p.pullSecret(ctx, secretPath); err != nil {
			return errors.WrapWithPath(err, "pull_secret", secretPath)
		}
		secretCount++
		logger.DebugCtx(ctx, "Pulled secret", "path", secretPath, "count", secretCount)
		fmt.Printf("✓ Pulled: %s\n", secretPath)
		return nil
	})

	if err != nil {
		logger.ErrorCtx(ctx, "Pull operation failed", 
			"error", err,
			"secrets_pulled", secretCount,
			"duration_ms", time.Since(start).Milliseconds())
		return err
	}

	logger.InfoCtx(ctx, "Pull operation completed successfully", 
		"secrets_pulled", secretCount,
		"duration_ms", time.Since(start).Milliseconds())
	
	fmt.Printf("\nSuccessfully pulled %d secrets\n", secretCount)
	return nil
}

func (p *Puller) pullSecret(ctx context.Context, secretPath string) error {
	start := time.Now()
	logger.DebugCtx(ctx, "Pulling secret", "path", secretPath)
	
	secret, err := p.client.ReadSecret(ctx, secretPath)
	if err != nil {
		return errors.WrapWithPath(err, "read_secret", secretPath)
	}

	localPath := p.getLocalPath(secretPath)
	logger.DebugCtx(ctx, "Writing to local file", "local_path", localPath)
	
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return errors.New("create_local_dir", err).
			WithContext("local_path", localPath).
			WithContext("secret_path", secretPath)
	}

	yamlData, err := yaml.Marshal(secret.Data)
	if err != nil {
		return errors.New("marshal_yaml", err).
			WithContext("secret_path", secretPath).
			WithContext("key_count", len(secret.Data))
	}

	if err := os.WriteFile(localPath, yamlData, 0600); err != nil {
		return errors.New("write_file", err).
			WithContext("local_path", localPath).
			WithContext("secret_path", secretPath)
	}

	logger.DebugCtx(ctx, "Successfully pulled secret", 
		"path", secretPath,
		"local_path", localPath,
		"key_count", len(secret.Data),
		"file_size", len(yamlData),
		"duration_ms", time.Since(start).Milliseconds())

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