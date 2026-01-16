package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
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
	if len(secretPath) >= 9 && secretPath[:9] == "kv/metadata" {
		dataPath = "kv/data" + secretPath[11:]
	}

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
			secretData, err := v.GetSecret(currentPath + "/" + key)
			if err != nil {
				fmt.Printf("Warning: Failed to get secret %s: %v\n", fullPath, err)
				continue
			}
			secrets[fullPath] = secretData
		}
	}

	return secrets, nil
}