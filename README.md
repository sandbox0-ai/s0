# s0 - Sandbox0 CLI

Command-line interface for Sandbox0.

## Installation

### Linux

```bash
# AMD64
curl -sLO https://github.com/sandbox0-ai/s0/releases/latest/download/s0-linux-amd64.tar.gz
tar -xzf s0-linux-amd64.tar.gz
chmod +x s0
sudo mv s0 /usr/local/bin/

# ARM64
curl -sLO https://github.com/sandbox0-ai/s0/releases/latest/download/s0-linux-arm64.tar.gz
tar -xzf s0-linux-arm64.tar.gz
chmod +x s0
sudo mv s0 /usr/local/bin/
```

### macOS

```bash
# Intel (AMD64)
curl -sLO https://github.com/sandbox0-ai/s0/releases/latest/download/s0-darwin-amd64.tar.gz
tar -xzf s0-darwin-amd64.tar.gz
chmod +x s0
sudo mv s0 /usr/local/bin/

# Apple Silicon (ARM64)
curl -sLO https://github.com/sandbox0-ai/s0/releases/latest/download/s0-darwin-arm64.tar.gz
tar -xzf s0-darwin-arm64.tar.gz
chmod +x s0
sudo mv s0 /usr/local/bin/
```

### Windows

Download `s0-windows-amd64.zip` or `s0-windows-arm64.zip` from [Releases](https://github.com/sandbox0-ai/s0/releases/latest), extract it, and add `s0.exe` to your PATH.

Using PowerShell:

```powershell
# AMD64
Invoke-WebRequest -Uri "https://github.com/sandbox0-ai/s0/releases/latest/download/s0-windows-amd64.zip" -OutFile "s0.zip"
Expand-Archive -Path "s0.zip" -DestinationPath "s0"
# Move s0.exe to a directory in your PATH
```

### Using Go (alternative)

If you have Go 1.21+ installed:

```bash
go install github.com/sandbox0-ai/s0/cmd/s0@latest
```

### Build from Source

```bash
git clone https://github.com/sandbox0-ai/s0.git
cd s0
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
s0 sandbox list
```

### Sandbox Files

```bash
s0 sandbox files ls [path] -s <sandbox-id>
s0 sandbox files cat <path> -s <sandbox-id>
s0 sandbox files stat <path> -s <sandbox-id>
s0 sandbox files mkdir <path> --parents -s <sandbox-id>
s0 sandbox files rm <path> -s <sandbox-id>
s0 sandbox files mv <src> <dst> -s <sandbox-id>
s0 sandbox files upload <local> <remote> -s <sandbox-id>
s0 sandbox files download <remote> <local> -s <sandbox-id>
s0 sandbox files write <path> --stdin|--data <content> -s <sandbox-id>
s0 sandbox files watch <path> --recursive -s <sandbox-id>
```

### Sandbox Context

```bash
s0 sandbox context list -s <sandbox-id>
s0 sandbox context get <ctx-id> -s <sandbox-id>
s0 sandbox context create --type repl|cmd --language <lang> --command <cmd> -s <sandbox-id>
s0 sandbox context delete <ctx-id> -s <sandbox-id>
s0 sandbox context restart <ctx-id> -s <sandbox-id>
s0 sandbox context input <ctx-id> <input> -s <sandbox-id>
s0 sandbox context signal <ctx-id> <signal> -s <sandbox-id>
s0 sandbox context stats <ctx-id> -s <sandbox-id>
```

### Sandbox Network

```bash
s0 sandbox network get -s <sandbox-id>
s0 sandbox network update --mode allow-all|block-all --allow-domain <domain> -s <sandbox-id>
```

### Sandbox Ports

```bash
s0 sandbox ports list -s <sandbox-id>
s0 sandbox ports expose <port> --resume -s <sandbox-id>
s0 sandbox ports unexpose <port> -s <sandbox-id>
s0 sandbox ports clear -s <sandbox-id>
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

### Template Image

```bash
s0 template image build [CONTEXT] -t <tag> [-f Dockerfile] [--platform linux/amd64]
s0 template image push <local-image> -t <target-tag>
```

## Examples

```bash
# Set token via environment variable
export SANDBOX0_TOKEN=your-token

# List templates
s0 template list

# Create a sandbox
s0 sandbox create --template my-template --ttl 3600

# List files in sandbox
s0 sandbox files ls /home/user -s <sandbox-id>

# Execute code in sandbox
s0 sandbox context create --type repl --language python -s <sandbox-id>
s0 sandbox context input <ctx-id> "print('hello')" -s <sandbox-id>

# Expose a port
s0 sandbox ports expose 8080 --resume -s <sandbox-id>

# Build and push a template image
s0 template image build . -t my-image:v1
s0 template image push my-image:v1 -t my-image:v1

# Create a volume and snapshot
s0 volume create
s0 snapshot create <volume-id> --name my-snapshot
```
