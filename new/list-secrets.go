package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hashicorp/vault-client-go"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("Usage: %s <namespace> <kvv2-path>\n", os.Args[0])
		fmt.Println("Example: go run list-secrets.go my-namespace kv/metadata")
		os.Exit(1)
	}

	namespace := os.Args[1]
	kvPath := os.Args[2]

	vaultAddr := os.Getenv("VAULT_ADDR")
	if vaultAddr == "" {
		log.Fatal("VAULT_ADDR environment variable is required")
	}
	fmt.Printf("Using Vault address: %s\n", vaultAddr)

	vaultToken := os.Getenv("VAULT_TOKEN")
	if vaultToken == "" {
		log.Fatal("VAULT_TOKEN environment variable is required")
	}

	client, err := vault.New(
		vault.WithAddress(vaultAddr),
		vault.WithRequestTimeout(60),
	)
	if err != nil {
		log.Fatal("Failed to create Vault client:", err)
	}

	if err := client.SetToken(vaultToken); err != nil {
		log.Fatal("Failed to set token:", err)
	}

	client.SetNamespace(namespace)

	// Test connection - skip health check for HCP Vault as it may not be accessible
	fmt.Println("Skipping health check for HCP Vault...")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Printf("Attempting to list secrets at path: %s in namespace: %s\n", kvPath, namespace)
	
	resp, err := client.Secrets.KvV2List(ctx, kvPath)
	if err != nil {
		log.Fatalf("Failed to list secrets: %v", err)
	}

	if resp.Data.Keys == nil || len(resp.Data.Keys) == 0 {
		fmt.Println("No secrets found at the specified path")
		return
	}

	fmt.Printf("Secrets at %s in namespace %s:\n", kvPath, namespace)
	for _, key := range resp.Data.Keys {
		fmt.Printf("  - %s\n", key)
	}
}