package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"vault-sync/internal/errors"
	"vault-sync/internal/logger"
	"vault-sync/internal/vault"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Test Vault connectivity and authentication",
	Long: `Test command to verify Vault connectivity, authentication, and KV mount accessibility.
This command will attempt to connect to Vault and list the root of the KV mount to validate configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		logger.InfoCtx(ctx, "Starting connectivity test")
		
		if err := cfg.Validate(); err != nil {
			return errors.Wrap(err, "validate_config")
		}

		client, err := vault.NewClient(cfg)
		if err != nil {
			return errors.Wrap(err, "create_vault_client")
		}

		fmt.Printf("✓ Successfully connected to Vault at %s\n", cfg.VaultAddr)
		if cfg.VaultNamespace != "" {
			fmt.Printf("✓ Using namespace: %s\n", cfg.VaultNamespace)
		}
		
		// Test listing the root of the KV mount
		logger.InfoCtx(ctx, "Testing KV mount accessibility", "mount", cfg.KVMount)
		fmt.Printf("Testing KV v2 mount '%s'...\n", cfg.KVMount)
		
		secrets, err := client.ListSecrets(ctx, cfg.BasePath)
		if err != nil {
			fmt.Printf("✗ Failed to list secrets at base path '%s'\n", cfg.BasePath)
			return errors.WrapWithPath(err, "test_list_secrets", cfg.BasePath)
		}
		
		fmt.Printf("✓ Successfully accessed KV mount '%s'\n", cfg.KVMount)
		if cfg.BasePath != "" {
			fmt.Printf("✓ Base path '%s' is accessible\n", cfg.BasePath)
		}
		
		fmt.Printf("Found %d items at the base path:\n", len(secrets))
		for i, secret := range secrets {
			if i >= 10 { // Limit output to first 10 items
				fmt.Printf("... and %d more\n", len(secrets)-10)
				break
			}
			fmt.Printf("  - %s\n", secret)
		}
		
		logger.InfoCtx(ctx, "Connectivity test completed successfully", 
			"mount", cfg.KVMount,
			"base_path", cfg.BasePath,
			"item_count", len(secrets))
		
		fmt.Printf("\n✓ All tests passed! Your Vault configuration is working correctly.\n")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(testCmd)
}