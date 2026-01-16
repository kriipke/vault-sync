package vault

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
	"vault-sync/internal/config"
	"vault-sync/internal/errors"
	"vault-sync/internal/logger"
)

type Client struct {
	client *vault.Client
	config *config.Config
}

type Secret struct {
	Path string            `yaml:"path"`
	Data map[string]string `yaml:"data"`
}

func NewClient(cfg *config.Config) (*Client, error) {
	logger.Debug("Creating Vault client", 
		"addr", cfg.VaultAddr,
		"namespace", cfg.VaultNamespace,
		"kv_mount", cfg.KVMount)
	
	client, err := vault.New(
		vault.WithAddress(cfg.VaultAddr),
		vault.WithRequestTimeout(30),
	)
	if err != nil {
		return nil, errors.New("create_vault_client", err).WithContext("vault_addr", cfg.VaultAddr)
	}

	if err := client.SetToken(cfg.VaultToken); err != nil {
		return nil, errors.New("set_vault_token", err)
	}

	if cfg.VaultNamespace != "" {
		client.SetNamespace(cfg.VaultNamespace)
		logger.Debug("Set Vault namespace", "namespace", cfg.VaultNamespace)
	}

	logger.Info("Successfully created Vault client")
	return &Client{
		client: client,
		config: cfg,
	}, nil
}

func (c *Client) ListSecrets(ctx context.Context, path string) ([]string, error) {
	start := time.Now()
	listPath := path
	if path != "" {
		listPath = strings.TrimPrefix(path, "/")
	}

	// Validate path format
	if err := c.validatePath(listPath); err != nil {
		return nil, errors.NewWithPath("validate_path", listPath, err).
			WithContext("mount", c.config.KVMount)
	}

	logger.DebugCtx(ctx, "Listing secrets", "path", listPath, "mount", c.config.KVMount)

	resp, err := c.client.Secrets.KvV2List(ctx, listPath, vault.WithMountPath(c.config.KVMount))
	if err != nil {
		vaultErr := errors.NewWithPath("list_secrets", listPath, err).
			WithContext("mount", c.config.KVMount).
			WithContext("duration_ms", time.Since(start).Milliseconds())
		
		// Add more context for different error types
		if responseErr, ok := err.(*vault.ResponseError); ok {
			vaultErr = vaultErr.
				WithContext("status_code", responseErr.StatusCode).
				WithContext("vault_errors", responseErr.Errors)
			
			logger.ErrorCtx(ctx, "Vault API error details", 
				"path", listPath,
				"status_code", responseErr.StatusCode,
				"vault_errors", responseErr.Errors)
			
			// Provide helpful hints based on status code
			switch responseErr.StatusCode {
			case http.StatusNotFound:
				vaultErr = vaultErr.WithContext("hint", "Path does not exist or insufficient permissions")
			case http.StatusForbidden:
				vaultErr = vaultErr.WithContext("hint", "Access denied - check token permissions")
			case http.StatusUnauthorized:
				vaultErr = vaultErr.WithContext("hint", "Authentication failed - check token validity")
			case http.StatusBadRequest:
				vaultErr = vaultErr.WithContext("hint", "Invalid request - check mount path and KV version")
			}
		}
		
		return nil, vaultErr
	}

	var secrets []string
	if resp != nil && resp.Data.Keys != nil {
		for _, key := range resp.Data.Keys {
			fullPath := path
			if fullPath == "" {
				fullPath = key
			} else {
				fullPath = strings.TrimSuffix(fullPath, "/") + "/" + key
			}
			secrets = append(secrets, fullPath)
		}
	}

	logger.DebugCtx(ctx, "Listed secrets successfully", 
		"path", listPath,
		"count", len(secrets),
		"duration_ms", time.Since(start).Milliseconds())

	return secrets, nil
}

func (c *Client) ReadSecret(ctx context.Context, secretPath string) (*Secret, error) {
	start := time.Now()
	readPath := strings.TrimPrefix(secretPath, "/")
	
	logger.DebugCtx(ctx, "Reading secret", "path", secretPath, "mount", c.config.KVMount)

	resp, err := c.client.Secrets.KvV2Read(ctx, readPath, vault.WithMountPath(c.config.KVMount))
	if err != nil {
		vaultErr := errors.NewWithPath("read_secret", secretPath, err).
			WithContext("mount", c.config.KVMount).
			WithContext("duration_ms", time.Since(start).Milliseconds())
		
		if responseErr, ok := err.(*vault.ResponseError); ok {
			vaultErr = vaultErr.
				WithContext("status_code", responseErr.StatusCode).
				WithContext("vault_errors", responseErr.Errors)
			
			logger.ErrorCtx(ctx, "Vault read error", 
				"path", secretPath,
				"status_code", responseErr.StatusCode,
				"vault_errors", responseErr.Errors)
		}
		
		return nil, vaultErr
	}

	data := make(map[string]string)
	if resp != nil && resp.Data.Data != nil {
		for k, v := range resp.Data.Data {
			if str, ok := v.(string); ok {
				data[k] = str
			} else {
				data[k] = fmt.Sprintf("%v", v)
			}
		}
	}

	logger.DebugCtx(ctx, "Read secret successfully", 
		"path", secretPath,
		"key_count", len(data),
		"duration_ms", time.Since(start).Milliseconds())

	return &Secret{
		Path: secretPath,
		Data: data,
	}, nil
}

func (c *Client) WriteSecret(ctx context.Context, secret *Secret) error {
	start := time.Now()
	writePath := strings.TrimPrefix(secret.Path, "/")
	
	logger.DebugCtx(ctx, "Writing secret", 
		"path", secret.Path, 
		"mount", c.config.KVMount,
		"key_count", len(secret.Data))

	secretData := make(map[string]interface{})
	for k, v := range secret.Data {
		secretData[k] = v
	}

	writeReq := schema.KvV2WriteRequest{
		Data: secretData,
	}

	_, err := c.client.Secrets.KvV2Write(ctx, writePath, writeReq, vault.WithMountPath(c.config.KVMount))
	if err != nil {
		vaultErr := errors.NewWithPath("write_secret", secret.Path, err).
			WithContext("mount", c.config.KVMount).
			WithContext("key_count", len(secret.Data)).
			WithContext("duration_ms", time.Since(start).Milliseconds())
		
		if responseErr, ok := err.(*vault.ResponseError); ok {
			vaultErr = vaultErr.
				WithContext("status_code", responseErr.StatusCode).
				WithContext("vault_errors", responseErr.Errors)
			
			logger.ErrorCtx(ctx, "Vault write error", 
				"path", secret.Path,
				"status_code", responseErr.StatusCode,
				"vault_errors", responseErr.Errors)
		}
		
		return vaultErr
	}

	logger.InfoCtx(ctx, "Wrote secret successfully", 
		"path", secret.Path,
		"key_count", len(secret.Data),
		"duration_ms", time.Since(start).Milliseconds())

	return nil
}

func (c *Client) WalkSecrets(ctx context.Context, basePath string, fn func(secretPath string) error) error {
	return c.walkSecretsRecursive(ctx, basePath, fn)
}

func (c *Client) validatePath(path string) error {
	// Check for common path issues
	if strings.Contains(path, "//") {
		return fmt.Errorf("path contains double slashes: consider using '%s'", strings.ReplaceAll(path, "//", "/"))
	}
	
	// Check for trailing slashes in non-directory contexts
	if path != "" && strings.HasSuffix(path, "/") && !strings.Contains(path, "/") {
		return fmt.Errorf("path has trailing slash but appears to be a secret name: consider '%s'", strings.TrimSuffix(path, "/"))
	}
	
	return nil
}

func (c *Client) walkSecretsRecursive(ctx context.Context, currentPath string, fn func(secretPath string) error) error {
	logger.DebugCtx(ctx, "Walking secrets recursively", "path", currentPath)
	
	secrets, err := c.ListSecrets(ctx, currentPath)
	if err != nil {
		return errors.WrapWithPath(err, "walk_secrets", currentPath)
	}

	for _, secretPath := range secrets {
		fullPath := secretPath
		if currentPath != "" {
			fullPath = path.Join(currentPath, secretPath)
		}

		if strings.HasSuffix(secretPath, "/") {
			logger.DebugCtx(ctx, "Descending into directory", "path", fullPath)
			if err := c.walkSecretsRecursive(ctx, strings.TrimSuffix(fullPath, "/"), fn); err != nil {
				return errors.WrapWithPath(err, "walk_secrets_recursive", fullPath)
			}
		} else {
			logger.DebugCtx(ctx, "Processing secret", "path", fullPath)
			if err := fn(fullPath); err != nil {
				return errors.WrapWithPath(err, "process_secret", fullPath)
			}
		}
	}

	return nil
}