package cmd

import (
	"context"

	"github.com/spf13/cobra"
	"vault-sync/internal/errors"
	"vault-sync/internal/logger"
	"vault-sync/internal/push"
	"vault-sync/internal/vault"
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push local YAML files to Vault",
	Long: `Reads local YAML files and pushes changes back to Vault KV v2.
For each secret, it fetches the current value from Vault, produces a human-readable
diff, and prompts the user for approval before writing (unless --yes is used).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		
		// Get flag values
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		autoApprove, _ := cmd.Flags().GetBool("yes")
		
		cfg.DryRun = dryRun
		cfg.AutoApprove = autoApprove
		
		logger.InfoCtx(ctx, "Starting push command", 
			"dry_run", dryRun, 
			"auto_approve", autoApprove)

		if err := cfg.Validate(); err != nil {
			return errors.Wrap(err, "validate_config")
		}

		client, err := vault.NewClient(cfg)
		if err != nil {
			return errors.Wrap(err, "create_vault_client")
		}

		pusher := push.New(client, cfg)
		
		return pusher.Push(ctx)
	},
}

func init() {
	pushCmd.Flags().Bool("dry-run", false, "Show diffs without writing to Vault")
	pushCmd.Flags().Bool("yes", false, "Auto-approve all changes without prompting")
	
	rootCmd.AddCommand(pushCmd)
}