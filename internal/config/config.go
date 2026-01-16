package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

type Config struct {
	VaultAddr      string
	VaultToken     string
	VaultNamespace string
	KVMount        string
	BasePath       string
	OutputDir      string
	DryRun         bool
	AutoApprove    bool
	Verbose        bool
	LogLevel       slog.Level
}

func New() *Config {
	homeDir, _ := os.UserHomeDir()
	
	return &Config{
		VaultAddr:      getEnvOrDefault("VAULT_ADDR", "http://localhost:8200"),
		VaultToken:     getEnvOrDefault("VAULT_TOKEN", ""),
		VaultNamespace: getEnvOrDefault("VAULT_NAMESPACE", ""),
		KVMount:        "kv",
		BasePath:       "",
		OutputDir:      filepath.Join(homeDir, ".vault-sync"),
		DryRun:         false,
		AutoApprove:    false,
		Verbose:        false,
		LogLevel:       slog.LevelInfo,
	}
}

func (c *Config) Validate() error {
	if c.VaultAddr == "" {
		return fmt.Errorf("vault address is required")
	}
	if c.VaultToken == "" {
		return fmt.Errorf("vault token is required")
	}
	if c.KVMount == "" {
		return fmt.Errorf("KV mount is required")
	}
	if c.OutputDir == "" {
		return fmt.Errorf("output directory is required")
	}
	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}