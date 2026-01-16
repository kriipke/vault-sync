package vault

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
	"vault-sync/internal/config"
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
	client, err := vault.New(
		vault.WithAddress(cfg.VaultAddr),
		vault.WithRequestTimeout(30),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	if err := client.SetToken(cfg.VaultToken); err != nil {
		return nil, fmt.Errorf("failed to set vault token: %w", err)
	}

	if cfg.VaultNamespace != "" {
		client.SetNamespace(cfg.VaultNamespace)
	}

	return &Client{
		client: client,
		config: cfg,
	}, nil
}

func (c *Client) ListSecrets(ctx context.Context, path string) ([]string, error) {
	listPath := path
	if path != "" {
		listPath = strings.TrimPrefix(path, "/")
	}

	resp, err := c.client.Secrets.KvV2List(ctx, listPath, vault.WithMountPath(c.config.KVMount))
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets at %s: %w", listPath, err)
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

	return secrets, nil
}

func (c *Client) ReadSecret(ctx context.Context, secretPath string) (*Secret, error) {
	readPath := strings.TrimPrefix(secretPath, "/")
	
	resp, err := c.client.Secrets.KvV2Read(ctx, readPath, vault.WithMountPath(c.config.KVMount))
	if err != nil {
		return nil, fmt.Errorf("failed to read secret at %s: %w", secretPath, err)
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

	return &Secret{
		Path: secretPath,
		Data: data,
	}, nil
}

func (c *Client) WriteSecret(ctx context.Context, secret *Secret) error {
	writePath := strings.TrimPrefix(secret.Path, "/")
	
	secretData := make(map[string]interface{})
	for k, v := range secret.Data {
		secretData[k] = v
	}

	writeReq := schema.KvV2WriteRequest{
		Data: secretData,
	}

	_, err := c.client.Secrets.KvV2Write(ctx, writePath, writeReq, vault.WithMountPath(c.config.KVMount))
	if err != nil {
		return fmt.Errorf("failed to write secret at %s: %w", secret.Path, err)
	}

	return nil
}

func (c *Client) WalkSecrets(ctx context.Context, basePath string, fn func(secretPath string) error) error {
	return c.walkSecretsRecursive(ctx, basePath, fn)
}

func (c *Client) walkSecretsRecursive(ctx context.Context, currentPath string, fn func(secretPath string) error) error {
	secrets, err := c.ListSecrets(ctx, currentPath)
	if err != nil {
		return err
	}

	for _, secretPath := range secrets {
		fullPath := secretPath
		if currentPath != "" {
			fullPath = path.Join(currentPath, secretPath)
		}

		if strings.HasSuffix(secretPath, "/") {
			if err := c.walkSecretsRecursive(ctx, strings.TrimSuffix(fullPath, "/"), fn); err != nil {
				return err
			}
		} else {
			if err := fn(fullPath); err != nil {
				return err
			}
		}
	}

	return nil
}