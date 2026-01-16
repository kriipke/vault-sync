package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type VaultClient struct {
	Address   string
	Token     string
	Namespace string
	client    *http.Client
}

type VaultListResponse struct {
	Data struct {
		Keys []string `json:"keys"`
	} `json:"data"`
}

type VaultSecretResponse struct {
	Data struct {
		Data     map[string]interface{} `json:"data"`
		Metadata struct {
			Version int `json:"version"`
		} `json:"metadata"`
	} `json:"data"`
}

func NewVaultClient(address, token, namespace string) *VaultClient {
	return &VaultClient{
		Address:   address,
		Token:     token,
		Namespace: namespace,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (v *VaultClient) ListSecrets(kvPath string) ([]string, error) {
	url := fmt.Sprintf("%s/v1/%s?list=true", v.Address, kvPath)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Vault-Token", v.Token)
	req.Header.Set("X-Vault-Namespace", v.Namespace)

	resp, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var vaultResp VaultListResponse
	if err := json.Unmarshal(body, &vaultResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return vaultResp.Data.Keys, nil
}

func (v *VaultClient) GetSecret(secretPath string) (map[string]interface{}, error) {
	// Convert metadata path to data path for KVv2
	dataPath := secretPath
	if len(secretPath) >= 11 && secretPath[:11] == "kv/metadata" {
		dataPath = "kv/data" + secretPath[11:]
	}

	fmt.Printf("Debug: Getting secret from path: %s (converted to: %s)\n", secretPath, dataPath)
	url := fmt.Sprintf("%s/v1/%s", v.Address, dataPath)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Vault-Token", v.Token)
	req.Header.Set("X-Vault-Namespace", v.Namespace)

	resp, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var vaultResp VaultSecretResponse
	if err := json.Unmarshal(body, &vaultResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return vaultResp.Data.Data, nil
}

func (v *VaultClient) PullSecretsRecursively(basePath string) (map[string]map[string]interface{}, error) {
	secrets := make(map[string]map[string]interface{})
	
	return v.pullSecretsRecursivelyHelper(basePath, secrets)
}

func (v *VaultClient) pullSecretsRecursivelyHelper(currentPath string, secrets map[string]map[string]interface{}) (map[string]map[string]interface{}, error) {
	keys, err := v.ListSecrets(currentPath)
	if err != nil {
		return secrets, fmt.Errorf("failed to list secrets at %s: %w", currentPath, err)
	}

	for _, key := range keys {
		fullPath := currentPath + "/" + key
		
		// If key ends with /, it's a folder - recurse into it
		if key[len(key)-1] == '/' {
			folderPath := currentPath + "/" + key[:len(key)-1] + "/metadata"
			secrets, err = v.pullSecretsRecursivelyHelper(folderPath, secrets)
			if err != nil {
				return secrets, err
			}
		} else {
			// It's a secret - fetch its data
			// Build correct path for secret data
			secretPath := currentPath + "/" + key
			// Convert from metadata path to data path
			if len(currentPath) >= 11 && currentPath[:11] == "kv/metadata" {
				secretPath = "kv/data" + currentPath[11:] + "/" + key
			}
			
			secretData, err := v.GetSecret(secretPath)
			if err != nil {
				fmt.Printf("Warning: Failed to get secret %s: %v\n", fullPath, err)
				continue
			}
			secrets[fullPath] = secretData
		}
	}

	return secrets, nil
}

func (v *VaultClient) PullSecretsToFiles(basePath, outputDir string) error {
	secrets, err := v.PullSecretsRecursively(basePath)
	if err != nil {
		return fmt.Errorf("failed to pull secrets: %w", err)
	}

	for secretPath, secretData := range secrets {
		if err := v.writeSecretToFile(secretPath, secretData, basePath, outputDir); err != nil {
			fmt.Printf("Warning: Failed to write secret %s: %v\n", secretPath, err)
		}
	}

	return nil
}

func (v *VaultClient) writeSecretToFile(secretPath string, secretData map[string]interface{}, basePath, outputDir string) error {
	// Convert vault path to file path
	// Remove the base path prefix and convert to relative file path
	relativePath := strings.TrimPrefix(secretPath, basePath+"/")
	
	// Remove kv/metadata or kv/data prefix if present
	if strings.HasPrefix(relativePath, "kv/metadata/") {
		relativePath = strings.TrimPrefix(relativePath, "kv/metadata/")
	} else if strings.HasPrefix(relativePath, "kv/data/") {
		relativePath = strings.TrimPrefix(relativePath, "kv/data/")
	}
	
	// Create file path with .yaml extension
	filePath := filepath.Join(outputDir, relativePath+".yaml")
	
	// Create directory structure
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Convert to YAML
	yamlData, err := yaml.Marshal(secretData)
	if err != nil {
		return fmt.Errorf("failed to convert to YAML: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	fmt.Printf("Written: %s\n", filePath)
	return nil
}