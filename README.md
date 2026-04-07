# s0 - Sandbox0 CLI

Command-line interface for Sandbox0.

## Installation

### One-step install

macOS and Linux:

```bash
curl -fsSL https://raw.githubusercontent.com/sandbox0-ai/s0/main/scripts/install.sh | bash
```

Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/sandbox0-ai/s0/main/scripts/install.ps1 | iex
```

Using Go:

```bash
go install github.com/sandbox0-ai/s0/cmd/s0@latest
```

### Release archives

Release archives are published at:

https://github.com/sandbox0-ai/s0/releases/latest

Archive names:

```text
s0-linux-amd64.tar.gz
s0-linux-arm64.tar.gz
s0-darwin-amd64.tar.gz
s0-darwin-arm64.tar.gz
s0-windows-amd64.zip
s0-windows-arm64.zip
```

### Manual install from release archives

#### Linux

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

#### macOS

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

#### Windows

Download `s0-windows-amd64.zip` or `s0-windows-arm64.zip` from [Releases](https://github.com/sandbox0-ai/s0/releases/latest), extract it, and add `s0.exe` to your PATH.

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
    gateway-mode: direct
    current-team-id: team_123
    token: ${SANDBOX0_TOKEN}
output:
  format: table
```

`gateway-mode` supports:

- `direct`: `api-url` is the working control-plane entrypoint.
- `global`: `api-url` is a Global Gateway entrypoint and workload commands are routed through the locally selected current team's home region.

Mode resolution order:

1. Explicit `gateway-mode` in the profile
2. `GET /metadata` returned by the API entrypoint
3. Fallback to `direct`

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SANDBOX0_TOKEN` | API authentication token | - |
| `SANDBOX0_BASE_URL` | API base URL | `https://api.sandbox0.ai` |

## GitHub Actions

`s0 template image build` and `s0 template image push` shell out to Docker.
Use a GitHub-hosted runner with Docker available, or a self-hosted runner with a working Docker daemon.

For CI, prefer a Sandbox0 API key scoped to automation. For image pushes, the recommended team role is `builder`.

### Setup Action

Install `s0`, add it to `PATH`, and optionally export `SANDBOX0_TOKEN` and `SANDBOX0_BASE_URL`:

```yaml
jobs:
  verify-s0:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6

      - uses: sandbox0-ai/s0/.github/actions/setup-s0@main
        with:
          token: ${{ secrets.SANDBOX0_TOKEN }}
          api-url: ${{ secrets.SANDBOX0_BASE_URL }}

      - run: s0 template list
```

### Reusable Workflow

Build and push a template image with the official reusable workflow:

```yaml
jobs:
  template-image:
    uses: sandbox0-ai/s0/.github/workflows/template-image.yml@main
    with:
      api-url: ${{ vars.SANDBOX0_BASE_URL }}
      image-tag: my-app:${{ github.sha }}
      context: .
      dockerfile: Dockerfile
    secrets:
      sandbox0_token: ${{ secrets.SANDBOX0_TOKEN }}
```

Consume the pushed template image reference from workflow outputs:

```yaml
jobs:
  publish:
    uses: sandbox0-ai/s0/.github/workflows/template-image.yml@main
    with:
      image-tag: my-app:${{ github.sha }}
    secrets:
      sandbox0_token: ${{ secrets.SANDBOX0_TOKEN }}

  deploy:
    needs: publish
    runs-on: ubuntu-latest
    steps:
      - run: echo "${{ needs.publish.outputs.template-image }}"
```

Pin `@main` to a release tag after the first s0 release that includes these GitHub Actions assets.

### Next Steps

1. Merge the GitHub Actions changes into `main`.
2. Publish the next `s0` release tag, for example `v0.2.3`, through the normal `s0` release flow.
3. Replace `@main` in workflow files with that release tag:

```yaml
- uses: sandbox0-ai/s0/.github/actions/setup-s0@v0.2.3
- uses: sandbox0-ai/s0/.github/workflows/template-image.yml@v0.2.3
```

If you want a floating major tag later, add and maintain `v0` after the tagged release is published.

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

In `global` mode, `auth`, `user`, `team`, and `admin` commands stay on the configured entrypoint. Workload-facing commands such as `sandbox`, `template`, `volume`, `credential`, `apikey`, and registry credential flows use the locally selected current team and switch to the home-region gateway automatically.

If a global-gateway profile has no current team selected yet, create one if needed and then select it locally:

```bash
s0 team create --name <name> --home-region <region-id>
s0 team use <team-id>
```

## Commands

### Team

```bash
s0 team create --name <name> [--slug <slug>] [--home-region <region-id>]
s0 team use <team-id>
```

### Admin Region

```bash
s0 admin region list
s0 admin region get <region-id>
s0 admin region create --id <id> --regional-gateway-url <url> [--display-name <name>] [--metering-export-url <url>] [--enabled=true|false]
s0 admin region update <region-id> [--display-name <name>] [--regional-gateway-url <url>] [--metering-export-url <url>] [--enabled=true|false]
s0 admin region delete <region-id>
```

`s0 admin region ...` targets the Global Gateway region directory and requires a system-admin token. `--edge-gateway-url` is also accepted as an alias for `--regional-gateway-url`.

### Sandbox

```bash
s0 sandbox run <sandbox-id> <input> [--alias <alias>] [--context-id <ctx-id>]
s0 sandbox create -t <template-id> [-f sandbox-config.yaml] [--ttl 3600] [--hard-ttl 7200] [--mount <volume-id>:/absolute/path] [--wait-for-mounts] [--mount-wait-timeout-ms 45000]
s0 sandbox get <sandbox-id>
s0 sandbox update <sandbox-id> [-f sandbox-update.yaml] [--ttl 3600] [--hard-ttl 7200] [--auto-resume true|false]
s0 sandbox delete <sandbox-id>
s0 sandbox pause <sandbox-id>
s0 sandbox resume <sandbox-id>
s0 sandbox refresh <sandbox-id>
s0 sandbox status <sandbox-id>
s0 sandbox list [--status <status>] [--template-id <id>] [--paused true|false] [--limit 50] [--offset 0]
```

Bootstrap mounts can be requested as part of sandbox creation:

```bash
s0 volume create
s0 sandbox create -t default \
  --mount <volume-id>:/workspace/data \
  --wait-for-mounts \
  --mount-wait-timeout-ms 45000

# Or provide a full claim request file.
cat <<'EOF' > sandbox-claim.yaml
template: default
mounts:
  - sandboxvolume_id: <volume-id>
    mount_point: /workspace/data
wait_for_mounts: true
mount_wait_timeout_ms: 45000
config:
  ttl: 3600
EOF
s0 sandbox create -f sandbox-claim.yaml
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
s0 sandbox context create --type repl|cmd [--alias <name>] [--command <cmd>] [--cwd <dir>] [--env KEY=VALUE] [--wait] -s <sandbox-id>
s0 sandbox context delete <ctx-id> -s <sandbox-id>
s0 sandbox context restart <ctx-id> -s <sandbox-id>
s0 sandbox context exec <ctx-id> <input> -s <sandbox-id>
s0 sandbox context signal <ctx-id> <signal> -s <sandbox-id>
s0 sandbox context stats <ctx-id> -s <sandbox-id>
```

`s0 sandbox run` is the REPL-oriented convenience command. It preserves state by
reusing a matching running REPL context when there is exactly one match for the
requested alias. `s0 sandbox exec` remains the one-shot command path.

### Sandbox Network

```bash
s0 sandbox create -t <template-id> -f sandbox-config.yaml
s0 sandbox network get -s <sandbox-id>
s0 sandbox network update --mode allow-all|block-all [--allow-cidr <cidr>] [--allow-domain <domain>] [--allow-port <port[/proto]|start-end[/proto]>] [--deny-cidr <cidr>] [--deny-domain <domain>] [--deny-port <port[/proto]|start-end[/proto]>] [--traffic-rule '<json>'] [--credential-rule '<json>'] [--credential-binding '<json>'] -s <sandbox-id>
s0 sandbox network update --policy-file network.yaml -s <sandbox-id>

# Claim-time network policy via sandbox config file
cat <<'EOF' > sandbox-config.yaml
network:
  mode: block-all
  egress:
    trafficRules:
      - name: allow-github-api
        action: allow
        domains: [api.github.com]
        ports:
          - port: 443
            protocol: tcp
  credentialBindings:
    - ref: gh-token
      sourceRef: github-source
      projection:
        type: http_headers
        httpHeaders:
          headers:
            - name: Authorization
              valueTemplate: "Bearer {{token}}"
EOF
s0 sandbox create -t default -f sandbox-config.yaml

# Simple compatibility path: block all except HTTPS to GitHub
s0 sandbox network update --mode block-all \
  --allow-domain github.com \
  --allow-port 443/tcp \
  -s <sandbox-id>

# Recommended for complex policies: edit a YAML file and apply it directly
cat <<'EOF' > network.yaml
mode: block-all
egress:
  trafficRules:
    - name: allow-ssh
      action: allow
      appProtocols: [ssh]
      ports:
        - port: 22
          protocol: tcp
credentialBindings:
  - ref: gh-token
    sourceRef: github-source
    projection:
      type: http_headers
      httpHeaders:
        headers:
          - name: Authorization
            valueTemplate: "Bearer {{token}}"
EOF
s0 sandbox network update --policy-file network.yaml -s <sandbox-id>

# Script-oriented structured flags: allow SSH traffic first, then inject outbound auth for GitHub API
s0 sandbox network update --mode block-all \
  --traffic-rule '{"name":"allow-ssh","action":"allow","appProtocols":["ssh"],"ports":[{"port":22,"protocol":"tcp"}]}' \
  --credential-binding '{"ref":"gh-token","sourceRef":"github-source","projection":{"type":"http_headers","httpHeaders":{"headers":[{"name":"Authorization","valueTemplate":"Bearer {{token}}"}]}}}' \
  --credential-rule '{"name":"github-auth","credentialRef":"gh-token","protocol":"https","domains":["api.github.com"],"ports":[{"port":443,"protocol":"tcp"}]}' \
  -s <sandbox-id>
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
s0 volume create [--access-mode RWO|RWX] [--cache-size <size>] [--prefetch <count>] [--buffer-size <size>] [--writeback true|false]
s0 volume delete <volume-id> [--force]
```

### Volume Files

```bash
s0 volume files ls <volume-id> [path]
s0 volume files cat <volume-id> <path>
s0 volume files stat <volume-id> <path>
s0 volume files mkdir <volume-id> <path> [--parents]
s0 volume files rm <volume-id> <path>
s0 volume files mv <volume-id> <src> <dst>
s0 volume files upload <volume-id> <local> <remote>
s0 volume files download <volume-id> <remote> <local>
s0 volume files write <volume-id> <path> --stdin|--data <content>
s0 volume files watch <volume-id> <path> --recursive
```

### Volume Snapshot

```bash
s0 volume snapshot list <volume-id>
s0 volume snapshot get <volume-id> <snapshot-id>
s0 volume snapshot create <volume-id> -n <name> [-d <description>]
s0 volume snapshot delete <volume-id> <snapshot-id>
s0 volume snapshot restore <volume-id> <snapshot-id>
```

### Sandbox Volume

```bash
s0 sandbox volume mount --volume-id <volume-id> --path <mount-path> [--cache-size <size>] [--buffer-size <size>] [--prefetch <count>] [--writeback true|false] -s <sandbox-id>
s0 sandbox volume unmount --volume-id <volume-id> --session-id <session-id> -s <sandbox-id>
s0 sandbox volume status -s <sandbox-id>
```

### Template Image

```bash
s0 template image build [CONTEXT] -t <tag> [-f Dockerfile] [--platform linux/amd64] [--no-cache] [--pull]
s0 template image push <local-image> -t <target-tag>
```

## Examples

```bash
# Set token via environment variable
export SANDBOX0_TOKEN=your-token

# List templates
s0 template list

# Create a sandbox
s0 sandbox create -t my-template --ttl 3600

# List files in sandbox
s0 sandbox files ls /home/user -s <sandbox-id>

# Operate on volume files directly without mounting into a sandbox first
s0 volume files write <volume-id> /docs/readme.txt --data "Hello from s0"
s0 volume files cat <volume-id> /docs/readme.txt
s0 volume files watch <volume-id> /docs --recursive

# Execute code in sandbox
s0 sandbox context create --type repl --alias python -s <sandbox-id>
s0 sandbox context exec <ctx-id> "print('hello')" -s <sandbox-id>

# Expose a port
s0 sandbox ports expose 8080 --resume -s <sandbox-id>

# Build and push a template image
s0 template image build . -t my-image:v1
s0 template image push my-image:v1 -t my-image:v1

# Create a volume and snapshot
s0 volume create
s0 volume snapshot create <volume-id> -n my-snapshot
```
