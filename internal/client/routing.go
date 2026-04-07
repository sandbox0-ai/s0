package client

import (
	"context"
	"errors"
	"strings"

	"github.com/sandbox0-ai/s0/internal/config"
	sandbox0 "github.com/sandbox0-ai/sdk-go"
)

type RouteScope string

const (
	RouteScopeEntrypoint RouteScope = "entrypoint"
	RouteScopeHomeRegion RouteScope = "home-region"
)

type ResolvedTarget struct {
	BaseURL     string
	Token       string
	GatewayMode config.GatewayMode
}

type ResolveTargetOptions struct {
	BaseURL               string
	Token                 string
	ConfiguredGatewayMode config.GatewayMode
	CurrentTeamID         string
	CurrentTeamTarget     *config.CurrentTeamTarget
	Scope                 RouteScope
	UserAgent             string
}

var ErrCurrentTeamRequired = errors.New("current team is not set; run `s0 team use <team-id>`")
var ErrCurrentTeamTargetRequired = errors.New("current team region endpoint is not set; run `s0 team use <team-id>`")

func tokenUsesImplicitTeamSelection(token string) bool {
	return !strings.HasPrefix(strings.TrimSpace(token), "s0_")
}

// ResolveTarget resolves the correct API target for the current command scope.
func ResolveTarget(ctx context.Context, opts ResolveTargetOptions) (*ResolvedTarget, error) {
	mode := opts.ConfiguredGatewayMode
	if mode == "" {
		if detected, ok := discoverGatewayMode(ctx, opts.BaseURL, opts.UserAgent); ok {
			mode = detected
		} else {
			mode = config.GatewayModeDirect
		}
	}

	target := &ResolvedTarget{
		BaseURL:     opts.BaseURL,
		Token:       opts.Token,
		GatewayMode: mode,
	}
	if opts.Scope != RouteScopeHomeRegion {
		return target, nil
	}
	if mode == config.GatewayModeGlobal {
		if strings.TrimSpace(opts.CurrentTeamID) == "" {
			return target, nil
		}
		if opts.CurrentTeamTarget == nil || strings.TrimSpace(opts.CurrentTeamTarget.GatewayURL) == "" {
			return target, nil
		}

		return &ResolvedTarget{
			BaseURL:     opts.CurrentTeamTarget.GatewayURL,
			Token:       opts.Token,
			GatewayMode: mode,
		}, nil
	}
	if tokenUsesImplicitTeamSelection(opts.Token) && strings.TrimSpace(opts.CurrentTeamID) == "" {
		return nil, ErrCurrentTeamRequired
	}
	if mode != config.GatewayModeGlobal {
		return target, nil
	}
	return target, nil
}

func discoverGatewayMode(ctx context.Context, baseURL, userAgent string) (config.GatewayMode, bool) {
	client, err := newSDKClient(baseURL, "", userAgent)
	if err != nil {
		return "", false
	}

	metadataRes, err := client.API().MetadataGet(ctx)
	if err != nil {
		return "", false
	}

	metadata, ok := metadataRes.Data.Get()
	if !ok {
		return "", false
	}

	return config.ParseGatewayMode(string(metadata.GatewayMode))
}

func newSDKClient(baseURL, token, userAgent string) (*sandbox0.Client, error) {
	opts := []sandbox0.Option{
		sandbox0.WithBaseURL(baseURL),
		sandbox0.WithToken(token),
	}
	if strings.TrimSpace(userAgent) != "" {
		opts = append(opts, sandbox0.WithUserAgent(userAgent))
	}
	return sandbox0.NewClient(opts...)
}
