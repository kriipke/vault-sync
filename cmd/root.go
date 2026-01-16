package cmd

import (
	"log/slog"

	"github.com/spf13/cobra"
	"vault-sync/internal/config"
	"vault-sync/internal/logger"
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
	rootCmd.PersistentFlags().BoolVarP(&cfg.Verbose, "verbose", "v", cfg.Verbose, "Enable verbose logging")
	
	// Add log level flag
	rootCmd.PersistentFlags().Var(&logLevelFlag{&cfg.LogLevel}, "log-level", "Set log level (debug, info, warn, error)")
}

func initConfig() {
	// Initialize logger with current config
	logger.Init(cfg)
	logger.Debug("Logger initialized", "verbose", cfg.Verbose, "level", cfg.LogLevel)
}

// logLevelFlag implements pflag.Value for slog.Level
type logLevelFlag struct {
	level *slog.Level
}

func (f *logLevelFlag) String() string {
	switch *f.level {
	case slog.LevelDebug:
		return "debug"
	case slog.LevelInfo:
		return "info"
	case slog.LevelWarn:
		return "warn"
	case slog.LevelError:
		return "error"
	default:
		return "info"
	}
}

func (f *logLevelFlag) Set(value string) error {
	switch value {
	case "debug":
		*f.level = slog.LevelDebug
	case "info":
		*f.level = slog.LevelInfo
	case "warn":
		*f.level = slog.LevelWarn
	case "error":
		*f.level = slog.LevelError
	default:
		*f.level = slog.LevelInfo
	}
	return nil
}

func (f *logLevelFlag) Type() string {
	return "string"
}