package docker

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
)

// Pusher handles Docker image pushing.
type Pusher struct {
	client *client.Client
}

// PushOptions contains options for pushing images.
type PushOptions struct {
	SourceImage string
	TargetImage string
	Registry    string
	Username    string
	Password    string
	Progress    io.Writer
}

// NewPusher creates a new Docker image pusher.
func NewPusher() (*Pusher, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	return &Pusher{client: cli}, nil
}

// Push pushes a Docker image to a registry.
func (p *Pusher) Push(ctx context.Context, opts PushOptions) error {
	// Progress writer defaults to stdout
	progress := opts.Progress
	if progress == nil {
		progress = os.Stdout
	}

	// Tag the image if source and target differ
	if opts.SourceImage != opts.TargetImage && opts.TargetImage != "" {
		if err := p.client.ImageTag(ctx, opts.SourceImage, opts.TargetImage); err != nil {
			return fmt.Errorf("failed to tag image: %w", err)
		}
		opts.SourceImage = opts.TargetImage
	}

	// Create auth config
	authConfig := registry.AuthConfig{
		ServerAddress: opts.Registry,
		Username:      opts.Username,
		Password:      opts.Password,
	}

	// Encode auth config
	authBytes, err := json.Marshal(authConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal auth config: %w", err)
	}
	authStr := base64.URLEncoding.EncodeToString(authBytes)

	// Push the image
	pushResp, err := p.client.ImagePush(ctx, opts.SourceImage, image.PushOptions{
		RegistryAuth: authStr,
	})
	if err != nil {
		return fmt.Errorf("failed to push image: %w", err)
	}
	defer pushResp.Close()

	// Stream push output
	return p.streamPushResponse(pushResp, progress)
}

// streamPushResponse streams push response to the writer.
func (p *Pusher) streamPushResponse(body io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Bytes()

		// Parse JSON stream response
		var resp pushStreamResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			fmt.Fprintf(w, "%s\n", line)
			continue
		}

		if resp.Status != "" {
			fmt.Fprintf(w, "%s", resp.Status)
			if resp.Progress != "" {
				fmt.Fprintf(w, " %s", resp.Progress)
			}
			fmt.Fprintln(w)
		}
		if resp.Error != "" {
			return fmt.Errorf("push error: %s", resp.Error)
		}
	}
	return scanner.Err()
}

// pushStreamResponse represents a Docker push API stream response.
type pushStreamResponse struct {
	Status   string `json:"status,omitempty"`
	Progress string `json:"progress,omitempty"`
	Error    string `json:"error,omitempty"`
	ID       string `json:"id,omitempty"`
}
