package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sandbox0-ai/s0/internal/config"
	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
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
	Scope                 RouteScope
	UserAgent             string
}

type gatewayMetadataEnvelope struct {
	Success bool `json:"success"`
	Data    struct {
		GatewayMode string `json:"gateway_mode"`
		Service     string `json:"service"`
	} `json:"data"`
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
	if opts.Scope != RouteScopeHomeRegion || mode != config.GatewayModeGlobal {
		return target, nil
	}

	globalClient, err := newSDKClient(opts.BaseURL, opts.Token, opts.UserAgent)
	if err != nil {
		return nil, err
	}

	activeTeamRes, err := globalClient.API().TenantActiveGet(ctx, apispec.TenantActiveGetParams{})
	if err != nil {
		return nil, fmt.Errorf("resolve active team: %w", err)
	}

	activeTeamSuccess, ok := activeTeamRes.(*apispec.SuccessActiveTeamResponse)
	if !ok {
		return nil, fmt.Errorf("resolve active team: unexpected response type %T", activeTeamRes)
	}

	activeTeam, ok := activeTeamSuccess.Data.Get()
	if !ok {
		return nil, fmt.Errorf("resolve active team: missing response data")
	}

	regionTokenRes, err := globalClient.API().AuthRegionTokenPost(ctx, apispec.NewOptIssueRegionTokenRequest(apispec.IssueRegionTokenRequest{
		TeamID: apispec.NewOptString(activeTeam.TeamID),
	}))
	if err != nil {
		return nil, fmt.Errorf("issue region token: %w", err)
	}

	regionTokenSuccess, ok := regionTokenRes.(*apispec.SuccessIssueRegionTokenResponse)
	if !ok {
		return nil, fmt.Errorf("issue region token: unexpected response type %T", regionTokenRes)
	}

	regionToken, ok := regionTokenSuccess.Data.Get()
	if !ok {
		return nil, fmt.Errorf("issue region token: missing response data")
	}

	regionalGatewayURL, ok := regionToken.RegionalGatewayURL.Get()
	if !ok || strings.TrimSpace(regionalGatewayURL) == "" {
		return nil, fmt.Errorf("issue region token: missing regional gateway URL")
	}

	return &ResolvedTarget{
		BaseURL:     regionalGatewayURL,
		Token:       regionToken.Token,
		GatewayMode: mode,
	}, nil
}

func discoverGatewayMode(ctx context.Context, baseURL, userAgent string) (config.GatewayMode, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/metadata", nil)
	if err != nil {
		return "", false
	}
	if strings.TrimSpace(userAgent) != "" {
		req.Header.Set("User-Agent", userAgent)
	}

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return "", false
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", false
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false
	}

	var envelope gatewayMetadataEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return "", false
	}
	if !envelope.Success {
		return "", false
	}

	return config.ParseGatewayMode(envelope.Data.GatewayMode)
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
