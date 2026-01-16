# vault-sync

A production-grade CLI for syncing Vault KV v2 secrets to/from your local filesystem.

## Features

- **Bidirectional sync**: Pull secrets from Vault to local YAML files or push local changes back to Vault
- **Vault namespace support**: Works with Vault Enterprise namespaces
- **Human-readable diffs**: See exactly what changes before applying them
- **Interactive approval**: Approve changes individually or use `--yes` for batch operations
- **Dry-run mode**: Preview changes without making them
- **Production-ready**: Clean architecture with separate packages and comprehensive error handling

## Installation

### Dependencies

```bash
go mod tidy
```

### Build

```bash
go build -o vault-sync .
```

## Configuration

Configuration can be set via environment variables or CLI flags:

| Environment Variable | CLI Flag | Default | Description |
|---------------------|----------|---------|-------------|
| `VAULT_ADDR` | `--vault-addr` | `http://localhost:8200` | Vault server address |
| `VAULT_TOKEN` | `--vault-token` | | Vault authentication token |
| `VAULT_NAMESPACE` | `--vault-namespace` | | Vault namespace (Enterprise) |
| | `--kv-mount` | `kv` | KV v2 mount name |
| | `--base-path` | | Base path in Vault to sync from |
| | `--output-dir` | `~/.vault-sync` | Local directory to sync to |

## Usage

### Pull secrets from Vault

```bash
# Pull all secrets from default mount
./vault-sync pull

# Pull secrets from specific path
./vault-sync pull --base-path secret/myapp

# Pull to custom directory
./vault-sync pull --output-dir ./secrets

# With Vault namespace
./vault-sync pull --vault-namespace prod
```

### Push local changes to Vault

```bash
# Push with interactive approval
./vault-sync push

# Auto-approve all changes
./vault-sync push --yes

# Dry run (show diffs without writing)
./vault-sync push --dry-run

# Push from custom directory
./vault-sync push --output-dir ./secrets
```

### Example workflow

```bash
# Set up environment
export VAULT_ADDR="https://vault.example.com"
export VAULT_TOKEN="hvs.abc123..."

# Pull secrets to local files
./vault-sync pull --base-path secret/myapp --output-dir ./myapp-secrets

# Edit local YAML files as needed
vim ./myapp-secrets/database.yaml

# Preview changes
./vault-sync push --dry-run --output-dir ./myapp-secrets

# Apply changes
./vault-sync push --output-dir ./myapp-secrets
```

### Directory structure

Local secrets are stored as YAML files mirroring the Vault path structure:

```
~/.vault-sync/
├── database.yaml          # secret/database
├── api/
│   ├── keys.yaml          # secret/api/keys
│   └── config.yaml        # secret/api/config
└── apps/
    └── web/
        └── env.yaml       # secret/apps/web/env
```

### YAML format

Each secret is stored as a simple key-value YAML file:

```yaml
username: admin
password: secret123
host: db.example.com
port: "5432"
```

## Architecture

The project follows a clean architecture with separated concerns:

```
vault-sync/
├── main.go                    # Entry point
├── cmd/                       # CLI commands (Cobra)
│   ├── root.go               # Root command and global flags
│   ├── pull.go               # Pull command
│   └── push.go               # Push command
└── internal/
    ├── config/               # Configuration management
    │   └── config.go
    ├── vault/                # Vault client wrapper
    │   └── client.go
    ├── pull/                 # Pull logic
    │   └── pull.go
    ├── push/                 # Push logic
    │   └── push.go
    └── diff/                 # Diff utilities
        └── diff.go
```

## Security considerations

- Secrets are stored with `0600` permissions (owner read/write only)
- Never logs secret values
- Supports Vault token authentication
- Works with Vault namespaces for multi-tenant environments