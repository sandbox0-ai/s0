package client

import (
	"context"

	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// Client wraps the sandbox0 SDK client with additional functionality.
type Client struct {
	*sandbox0.Client
}

// NewClient creates a new Client with the given options.
func NewClient(opts ...sandbox0.Option) (*Client, error) {
	client, err := sandbox0.NewClient(opts...)
	if err != nil {
		return nil, err
	}
	return &Client{Client: client}, nil
}

// RegistryCredentials represents registry authentication credentials.
type RegistryCredentials struct {
	Provider  string
	Registry  string
	Username  string
	Password  string
	ExpiresAt string
}

// GetRegistryCredentials retrieves temporary registry credentials for image push.
func (c *Client) GetRegistryCredentials(ctx context.Context) (*RegistryCredentials, error) {
	resp, err := c.API().APIV1RegistryCredentialsPost(ctx)
	if err != nil {
		return nil, err
	}

	successResp, ok := resp.(*apispec.SuccessRegistryCredentialsResponse)
	if !ok {
		return nil, err
	}

	data, ok := successResp.Data.Get()
	if !ok {
		return nil, err
	}

	creds := &RegistryCredentials{
		Provider: data.Provider,
		Registry: data.Registry,
		Username: data.Username,
		Password: data.Password,
	}

	if expiresAt, ok := data.ExpiresAt.Get(); ok {
		creds.ExpiresAt = expiresAt.String()
	}

	return creds, nil
}
