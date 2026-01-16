package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"vault-sync/internal/pull"
	"vault-sync/internal/vault"
)

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull secrets from Vault to local filesystem",
	Long: `Recursively downloads secrets from Vault KV v2 and writes them as YAML files
to the local filesystem. The directory structure mirrors the Vault path structure.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("configuration error: %w", err)
		}

		client, err := vault.NewClient(cfg)
		if err != nil {
			return fmt.Errorf("failed to create Vault client: %w", err)
		}

		puller := pull.New(client, cfg)
		
		ctx := context.Background()
		return puller.Pull(ctx)
	},
}

func init() {
	rootCmd.AddCommand(pullCmd)
}