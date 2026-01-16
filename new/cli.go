package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <command> [args...]\n", os.Args[0])
		fmt.Println("Commands:")
		fmt.Println("  list <namespace> <kvv2-path>  List secret names")
		fmt.Println("  pull <namespace> <kvv2-path>  Pull all secrets recursively")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "list":
		handleListCommand()
	case "pull":
		handlePullCommand()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}

func handleListCommand() {
	if len(os.Args) < 4 {
		fmt.Printf("Usage: %s list <namespace> <kvv2-path>\n", os.Args[0])
		fmt.Println("Example: go run . list my-namespace kv/metadata")
		os.Exit(1)
	}

	namespace := os.Args[2]
	kvPath := os.Args[3]

	vaultAddr := os.Getenv("VAULT_ADDR")
	if vaultAddr == "" {
		log.Fatal("VAULT_ADDR environment variable is required")
	}

	vaultToken := os.Getenv("VAULT_TOKEN")
	if vaultToken == "" {
		log.Fatal("VAULT_TOKEN environment variable is required")
	}

	client := NewVaultClient(vaultAddr, vaultToken, namespace)

	secrets, err := client.ListSecrets(kvPath)
	if err != nil {
		log.Fatalf("Failed to list secrets: %v", err)
	}

	if len(secrets) == 0 {
		fmt.Println("No secrets found at the specified path")
		return
	}

	fmt.Printf("Secrets at %s in namespace %s:\n", kvPath, namespace)
	for _, secret := range secrets {
		fmt.Printf("  - %s\n", secret)
	}
}

func handlePullCommand() {
	if len(os.Args) < 4 {
		fmt.Printf("Usage: %s pull <namespace> <kvv2-path>\n", os.Args[0])
		fmt.Println("Example: go run . pull my-namespace kv/metadata")
		os.Exit(1)
	}

	namespace := os.Args[2]
	kvPath := os.Args[3]

	vaultAddr := os.Getenv("VAULT_ADDR")
	if vaultAddr == "" {
		log.Fatal("VAULT_ADDR environment variable is required")
	}

	vaultToken := os.Getenv("VAULT_TOKEN")
	if vaultToken == "" {
		log.Fatal("VAULT_TOKEN environment variable is required")
	}

	client := NewVaultClient(vaultAddr, vaultToken, namespace)

	fmt.Printf("Pulling all secrets recursively from %s in namespace %s...\n", kvPath, namespace)
	secrets, err := client.PullSecretsRecursively(kvPath)
	if err != nil {
		log.Fatalf("Failed to pull secrets: %v", err)
	}

	if len(secrets) == 0 {
		fmt.Println("No secrets found at the specified path")
		return
	}

	fmt.Printf("\nFound %d secrets:\n", len(secrets))
	for path, data := range secrets {
		fmt.Printf("\n--- %s ---\n", path)
		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			fmt.Printf("Error formatting secret: %v\n", err)
			continue
		}
		fmt.Println(string(jsonData))
	}
}