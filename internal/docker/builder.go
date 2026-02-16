package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/docker/docker/client"
)

// Builder handles Docker image building.
type Builder struct {
	client *client.Client
}

// BuildOptions contains options for building images.
type BuildOptions struct {
	Context    string
	Dockerfile string
	Tags       []string
	Platform   string
	BuildArgs  map[string]*string
	NoCache    bool
	Pull       bool
	Progress   io.Writer
}

// NewBuilder creates a new Docker image builder.
func NewBuilder() (*Builder, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	return &Builder{client: cli}, nil
}

// Build builds a Docker image from the given context using docker CLI.
func (b *Builder) Build(ctx context.Context, opts BuildOptions) error {
	// Default context to current directory
	if opts.Context == "" {
		opts.Context = "."
	}

	// Default Dockerfile
	if opts.Dockerfile == "" {
		opts.Dockerfile = "Dockerfile"
	}

	// Progress writer defaults to stdout
	progress := opts.Progress
	if progress == nil {
		progress = os.Stdout
	}

	// Build docker command arguments
	args := []string{"build"}
	args = append(args, "-f", opts.Dockerfile)

	for _, tag := range opts.Tags {
		args = append(args, "-t", tag)
	}

	if opts.Platform != "" {
		args = append(args, "--platform", opts.Platform)
	}

	for key, value := range opts.BuildArgs {
		if value != nil {
			args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, *value))
		}
	}

	if opts.NoCache {
		args = append(args, "--no-cache")
	}

	if opts.Pull {
		args = append(args, "--pull")
	}

	args = append(args, opts.Context)

	// Run docker build command
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = progress
	cmd.Stderr = progress

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.Canceled {
			return ctx.Err()
		}
		return fmt.Errorf("docker build failed: %w", err)
	}

	return nil
}

// IsAvailable checks if docker CLI is available.
func IsAvailable() bool {
	_, err := exec.LookPath("docker")
	return err == nil
}

// Info returns Docker version info.
func Info() (string, error) {
	cmd := exec.Command("docker", "version", "--format", "{{.Server.Version}}")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get docker version: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
