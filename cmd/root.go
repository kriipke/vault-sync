package cmd

import (
	"github.com/spf13/cobra"
	"vault-sync/internal/config"
)

var cfg *config.Config

var rootCmd = &cobra.Command{
	Use:   "vault-sync",
	Short: "A production-grade CLI for syncing Vault KV v2 secrets",
	Long: `vault-sync is a CLI tool that allows you to synchronize Vault KV v2 secrets
with your local filesystem. It supports bidirectional sync with pull and push operations,
Vault namespaces, and provides human-readable diffs for changes.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Initialize config first so we can bind flags to it
	cfg = config.New()

	rootCmd.PersistentFlags().StringVar(&cfg.VaultAddr, "vault-addr", cfg.VaultAddr, "Vault server address (default: $VAULT_ADDR or http://localhost:8200)")
	rootCmd.PersistentFlags().StringVar(&cfg.VaultToken, "vault-token", cfg.VaultToken, "Vault authentication token (default: $VAULT_TOKEN)")
	rootCmd.PersistentFlags().StringVar(&cfg.VaultNamespace, "vault-namespace", cfg.VaultNamespace, "Vault namespace (default: $VAULT_NAMESPACE)")
	rootCmd.PersistentFlags().StringVar(&cfg.KVMount, "kv-mount", cfg.KVMount, "KV v2 mount name")
	rootCmd.PersistentFlags().StringVar(&cfg.BasePath, "base-path", cfg.BasePath, "Base path in Vault to sync from")
	rootCmd.PersistentFlags().StringVar(&cfg.OutputDir, "output-dir", cfg.OutputDir, "Local directory to sync to (default: ~/.vault-sync)")
}

func initConfig() {
	// Config is already initialized in init()
}