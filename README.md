# s0 - Sandbox0 CLI

Command-line interface for Sandbox0.

## Installation

```bash
make build
cp bin/s0 /usr/local/bin/
```

## Configuration

Configuration file: `~/.s0/config.yaml`

```yaml
current-profile: default
profiles:
  default:
    api-url: https://api.sandbox0.ai
    token: ${SANDBOX0_TOKEN}
output:
  format: table
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SANDBOX0_TOKEN` | API authentication token | - |
| `SANDBOX0_BASE_URL` | API base URL | `https://api.sandbox0.ai` |

## Usage

```bash
# Global flags
s0 [flags] [command]

Flags:
  --api-url string   Override API URL
  -c, --config string   Config file (default ~/.s0/config.yaml)
  -o, --output string   Output format: table|json|yaml (default "table")
  -p, --profile string  Profile name (default "default")
  --token string     Override API token
```

## Commands

### Sandbox

```bash
s0 sandbox create --template <id> [--ttl 3600] [--hard-ttl 7200]
s0 sandbox get <sandbox-id>
s0 sandbox delete <sandbox-id>
s0 sandbox pause <sandbox-id>
s0 sandbox resume <sandbox-id>
s0 sandbox refresh <sandbox-id>
s0 sandbox status <sandbox-id>
```

### Template

```bash
s0 template list
s0 template get <template-id>
s0 template create --id <id> --spec-file template.yaml
s0 template update <template-id> --spec-file template.yaml
s0 template delete <template-id>
```

### Volume

```bash
s0 volume list
s0 volume get <volume-id>
s0 volume create
s0 volume delete <volume-id>
```

### Snapshot

```bash
s0 snapshot list <volume-id>
s0 snapshot get <volume-id> <snapshot-id>
s0 snapshot create <volume-id> --name <name>
s0 snapshot delete <volume-id> <snapshot-id>
s0 snapshot restore <volume-id> <snapshot-id>
```

### Image

```bash
s0 image build [CONTEXT] -t <tag> [-f Dockerfile] [--platform linux/amd64]
s0 image push <local-image> -t <target-tag>
s0 image credentials    # Show registry credentials
```

## Examples

```bash
# Set token via environment variable
export SANDBOX0_TOKEN=your-token

# List templates
s0 template list

# Create a sandbox
s0 sandbox create --template my-template --ttl 3600

# Build and push an image
s0 image build . -t my-image:v1
s0 image push my-image:v1 -t my-image:v1

# Create a volume and snapshot
s0 volume create
s0 snapshot create <volume-id> --name my-snapshot
```
