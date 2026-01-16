package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
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
		// Get flag values
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		autoApprove, _ := cmd.Flags().GetBool("yes")
		
		cfg.DryRun = dryRun
		cfg.AutoApprove = autoApprove

		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("configuration error: %w", err)
		}

		client, err := vault.NewClient(cfg)
		if err != nil {
			return fmt.Errorf("failed to create Vault client: %w", err)
		}

		pusher := push.New(client, cfg)
		
		ctx := context.Background()
		return pusher.Push(ctx)
	},
}

func init() {
	pushCmd.Flags().Bool("dry-run", false, "Show diffs without writing to Vault")
	pushCmd.Flags().Bool("yes", false, "Auto-approve all changes without prompting")
	
	rootCmd.AddCommand(pushCmd)
}