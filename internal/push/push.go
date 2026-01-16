package push

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"vault-sync/internal/config"
	"vault-sync/internal/diff"
	"vault-sync/internal/errors"
	"vault-sync/internal/logger"
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
	start := time.Now()
	logger.InfoCtx(ctx, "Starting push operation", 
		"output_dir", p.config.OutputDir,
		"dry_run", p.config.DryRun,
		"auto_approve", p.config.AutoApprove)
	
	fmt.Printf("Pushing secrets from %s to Vault\n", p.config.OutputDir)
	
	if _, err := os.Stat(p.config.OutputDir); os.IsNotExist(err) {
		return errors.New("output_dir_not_found", err).
			WithContext("output_dir", p.config.OutputDir)
	}

	var localSecrets []*vault.Secret
	err := filepath.Walk(p.config.OutputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.WarnCtx(ctx, "Error walking file", "path", path, "error", err)
			return errors.New("walk_file", err).WithContext("path", path)
		}

		if !info.IsDir() && strings.HasSuffix(path, ".yaml") {
			logger.DebugCtx(ctx, "Loading local secret", "path", path)
			secret, err := p.loadLocalSecret(path)
			if err != nil {
				return errors.WrapWithPath(err, "load_local_secret", path)
			}
			localSecrets = append(localSecrets, secret)
		}

		return nil
	})

	if err != nil {
		return errors.Wrap(err, "walk_output_directory")
	}

	logger.InfoCtx(ctx, "Found local secrets", "count", len(localSecrets))

	pushCount := 0
	for i, localSecret := range localSecrets {
		logger.DebugCtx(ctx, "Processing secret", 
			"path", localSecret.Path, 
			"progress", fmt.Sprintf("%d/%d", i+1, len(localSecrets)))
		
		shouldPush, err := p.processSecret(ctx, localSecret)
		if err != nil {
			logger.ErrorCtx(ctx, "Failed to process secret", 
				"path", localSecret.Path, 
				"error", err)
			return errors.WrapWithPath(err, "process_secret", localSecret.Path)
		}

		if shouldPush {
			pushCount++
		}
	}

	logger.InfoCtx(ctx, "Push operation completed", 
		"total_secrets", len(localSecrets),
		"pushed_count", pushCount,
		"duration_ms", time.Since(start).Milliseconds())

	fmt.Printf("\nProcessed %d local secrets, pushed %d changes\n", len(localSecrets), pushCount)
	return nil
}

func (p *Pusher) processSecret(ctx context.Context, localSecret *vault.Secret) (bool, error) {
	start := time.Now()
	logger.DebugCtx(ctx, "Processing secret", "path", localSecret.Path)
	
	fmt.Printf("\nProcessing: %s\n", localSecret.Path)

	currentSecret, err := p.client.ReadSecret(ctx, localSecret.Path)
	if err != nil {
		logger.InfoCtx(ctx, "Secret does not exist in Vault, will create new", "path", localSecret.Path)
		fmt.Printf("Secret %s does not exist in Vault (will create new)\n", localSecret.Path)
		currentSecret = &vault.Secret{
			Path: localSecret.Path,
			Data: make(map[string]string),
		}
	}

	secretDiff, err := diff.CompareSecrets(currentSecret, localSecret)
	if err != nil {
		return false, errors.WrapWithPath(err, "compare_secrets", localSecret.Path)
	}

	if !secretDiff.HasDiff {
		logger.DebugCtx(ctx, "No changes needed", "path", localSecret.Path)
		fmt.Printf("✓ No changes needed for %s\n", localSecret.Path)
		return false, nil
	}

	logger.InfoCtx(ctx, "Changes detected for secret", 
		"path", localSecret.Path,
		"has_diff", secretDiff.HasDiff)
	
	fmt.Println("Changes detected:")
	diff.PrintDiff(secretDiff)

	if p.config.DryRun {
		logger.InfoCtx(ctx, "Dry run mode - would update secret", "path", localSecret.Path)
		fmt.Printf("✓ [DRY RUN] Would update %s\n", localSecret.Path)
		return false, nil
	}

	if !p.config.AutoApprove {
		if !p.promptForApproval(localSecret.Path) {
			logger.InfoCtx(ctx, "User skipped secret update", "path", localSecret.Path)
			fmt.Printf("✗ Skipped %s\n", localSecret.Path)
			return false, nil
		}
	}

	logger.InfoCtx(ctx, "Writing secret to Vault", "path", localSecret.Path)
	if err := p.client.WriteSecret(ctx, localSecret); err != nil {
		return false, errors.WrapWithPath(err, "write_secret", localSecret.Path)
	}

	logger.InfoCtx(ctx, "Successfully updated secret", 
		"path", localSecret.Path,
		"duration_ms", time.Since(start).Milliseconds())
	
	fmt.Printf("✓ Updated %s\n", localSecret.Path)
	return true, nil
}

func (p *Pusher) loadLocalSecret(filePath string) (*vault.Secret, error) {
	logger.Debug("Loading local secret file", "file_path", filePath)
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errors.New("read_file", err).WithContext("file_path", filePath)
	}

	var secretData map[string]string
	if err := yaml.Unmarshal(data, &secretData); err != nil {
		return nil, errors.New("parse_yaml", err).
			WithContext("file_path", filePath).
			WithContext("file_size", len(data))
	}

	vaultPath := p.getVaultPath(filePath)
	
	logger.Debug("Loaded local secret successfully", 
		"file_path", filePath,
		"vault_path", vaultPath,
		"key_count", len(secretData))
	
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