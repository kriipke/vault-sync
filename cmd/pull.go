package cmd

import (
	"context"

	"github.com/spf13/cobra"
	"vault-sync/internal/errors"
	"vault-sync/internal/logger"
	"vault-sync/internal/pull"
	"vault-sync/internal/vault"
)

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull secrets from Vault to local filesystem",
	Long: `Recursively downloads secrets from Vault KV v2 and writes them as YAML files
to the local filesystem. The directory structure mirrors the Vault path structure.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		logger.InfoCtx(ctx, "Starting pull command")
		
		if err := cfg.Validate(); err != nil {
			return errors.Wrap(err, "validate_config")
		}

		client, err := vault.NewClient(cfg)
		if err != nil {
			return errors.Wrap(err, "create_vault_client")
		}

		puller := pull.New(client, cfg)
		
		return puller.Pull(ctx)
	},
}

func init() {
	rootCmd.AddCommand(pullCmd)
}