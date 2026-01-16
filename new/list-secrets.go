package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("Usage: %s <namespace> <kvv2-path>\n", os.Args[0])
		fmt.Println("Example: go run list-secrets.go my-namespace secret/data")
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

	client, err := vault.New(
		vault.WithAddress(vaultAddr),
		vault.WithRequestTimeout(30),
	)
	if err != nil {
		log.Fatal("Failed to create Vault client:", err)
	}

	if err := client.SetToken(vaultToken); err != nil {
		log.Fatal("Failed to set token:", err)
	}

	client.SetNamespace(namespace)

	ctx := context.Background()

	resp, err := client.Secrets.KvV2List(ctx, kvPath)
	if err != nil {
		log.Fatal("Failed to list secrets:", err)
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