package docker

import (
	"archive/tar"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
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

// Build builds a Docker image from the given context.
func (b *Builder) Build(ctx context.Context, opts BuildOptions) error {
	// Default context to current directory
	if opts.Context == "" {
		opts.Context = "."
	}

	// Default Dockerfile
	if opts.Dockerfile == "" {
		opts.Dockerfile = "Dockerfile"
	}

	// Create build context tar
	buildContext, err := createBuildContext(opts.Context)
	if err != nil {
		return fmt.Errorf("failed to create build context: %w", err)
	}
	defer buildContext.Close()

	// Build options
	buildOpts := types.ImageBuildOptions{
		Dockerfile:  opts.Dockerfile,
		Tags:        opts.Tags,
		BuildArgs:   opts.BuildArgs,
		NoCache:     opts.NoCache,
		Remove:      true,
		ForceRemove: true,
		PullParent:  opts.Pull,
		Platform:    opts.Platform,
	}

	// Progress writer defaults to stdout
	progress := opts.Progress
	if progress == nil {
		progress = os.Stdout
	}

	resp, err := b.client.ImageBuild(ctx, buildContext, buildOpts)
	if err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}
	defer resp.Body.Close()

	// Stream build output
	return streamDockerResponse(resp.Body, progress)
}

// createBuildContext creates a tar archive of the build context.
func createBuildContext(contextPath string) (*os.File, error) {
	absPath, err := filepath.Abs(contextPath)
	if err != nil {
		return nil, err
	}

	// Check if .dockerignore exists and parse it
	// For simplicity, we'll use Docker's built-in context handling
	// by creating a basic tar of the directory

	tmpFile, err := os.CreateTemp("", "docker-build-context-*.tar")
	if err != nil {
		return nil, err
	}

	tw := tar.NewWriter(tmpFile)

	err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories (they're created implicitly)
		if info.IsDir() {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(absPath, path)
		if err != nil {
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		// Write header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// Write file content
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(tw, file)
		return err
	})

	if err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, err
	}

	if err := tw.Close(); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, err
	}

	// Seek to beginning for reading
	if _, err := tmpFile.Seek(0, 0); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, err
	}

	return tmpFile, nil
}

// streamDockerResponse streams Docker API response to the writer.
func streamDockerResponse(body io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Bytes()

		// Parse JSON stream response
		var resp dockerStreamResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			fmt.Fprintf(w, "%s\n", line)
			continue
		}

		if resp.Stream != "" {
			fmt.Fprint(w, resp.Stream)
		}
		if resp.Error != "" {
			return fmt.Errorf("build error: %s", resp.Error)
		}
		if resp.Status != "" {
			fmt.Fprintf(w, "%s", resp.Status)
			if resp.Progress != nil {
				fmt.Fprintf(w, " %s", resp.Progress)
			}
			fmt.Fprintln(w)
		}
	}
	return scanner.Err()
}

// dockerStreamResponse represents a Docker API stream response.
type dockerStreamResponse struct {
	Stream   string      `json:"stream,omitempty"`
	Status   string      `json:"status,omitempty"`
	Progress interface{} `json:"progressDetail,omitempty"`
	Error    string      `json:"error,omitempty"`
}
