package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type VaultListResponse struct {
	Data struct {
		Keys []string `json:"keys"`
	} `json:"data"`
}

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("Usage: %s <namespace> <kvv2-path>\n", os.Args[0])
		fmt.Println("Example: go run list-secrets-http.go my-namespace kv/metadata")
		os.Exit(1)
	}

	namespace := os.Args[1]
	kvPath := os.Args[2]

	vaultAddr := os.Getenv("VAULT_ADDR")
	if vaultAddr == "" {
		log.Fatal("VAULT_ADDR environment variable is required")
	}

	vaultToken := os.Getenv("VAULT_TOKEN")
	if vaultToken == "" {
		log.Fatal("VAULT_TOKEN environment variable is required")
	}

	url := fmt.Sprintf("%s/v1/%s?list=true", vaultAddr, kvPath)
	
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal("Failed to create request:", err)
	}

	req.Header.Set("X-Vault-Token", vaultToken)
	req.Header.Set("X-Vault-Namespace", namespace)

	fmt.Printf("Requesting: %s\n", url)
	fmt.Printf("Namespace: %s\n", namespace)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Request failed:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Failed to read response:", err)
	}

	var vaultResp VaultListResponse
	if err := json.Unmarshal(body, &vaultResp); err != nil {
		log.Fatal("Failed to parse JSON:", err)
	}

	if len(vaultResp.Data.Keys) == 0 {
		fmt.Println("No secrets found at the specified path")
		return
	}

	fmt.Printf("Secrets at %s in namespace %s:\n", kvPath, namespace)
	for _, key := range vaultResp.Data.Keys {
		fmt.Printf("  - %s\n", key)
	}
}