package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/sandbox0-ai/s0/internal/config"
	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

type authProvider struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Type                string `json:"type"`
	BrowserLoginEnabled bool   `json:"browser_login_enabled"`
	DeviceLoginEnabled  bool   `json:"device_login_enabled"`
}

type authLoginData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

type authEnvelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type authLoginMode string

const (
	authLoginModeAuto    authLoginMode = "auto"
	authLoginModeDevice  authLoginMode = "device"
	authLoginModeBuiltin authLoginMode = "builtin"

	staleSelectedTeamGrantHint = "token stale, run `s0 auth login` or refresh token, then run `s0 team use <team-id>` and retry"
)

func authRequest(ctx context.Context, method, endpoint, token string, requestBody any, responseData any) error {
	var body io.Reader
	if requestBody != nil {
		raw, err := json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		body = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	rawResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var envelope authEnvelope
	if err := json.Unmarshal(rawResp, &envelope); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if !envelope.Success {
		if envelope.Error != nil && envelope.Error.Message != "" {
			return fmt.Errorf("%s", envelope.Error.Message)
		}
		return fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	if responseData != nil && len(envelope.Data) > 0 {
		if err := json.Unmarshal(envelope.Data, responseData); err != nil {
			return fmt.Errorf("decode response data: %w", err)
		}
	}

	return nil
}

func fetchAuthProviders(ctx context.Context, baseURL string) ([]authProvider, error) {
	var data struct {
		Providers []authProvider `json:"providers"`
	}
	err := authRequest(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/auth/providers", "", nil, &data)
	if err != nil {
		return nil, err
	}
	return data.Providers, nil
}

func selectAuthProvider(providers []authProvider, requestedMode string) (*authProvider, authLoginMode, error) {
	mode := authLoginMode(strings.TrimSpace(strings.ToLower(requestedMode)))
	if mode == "" {
		mode = authLoginModeAuto
	}

	switch mode {
	case authLoginModeAuto:
		for i := range providers {
			provider := &providers[i]
			if provider.Type == "oidc" && provider.DeviceLoginEnabled {
				return provider, authLoginModeDevice, nil
			}
			if provider.Type == "builtin" {
				return provider, authLoginModeBuiltin, nil
			}
		}
		return nil, "", fmt.Errorf("no supported auth provider found")
	case authLoginModeDevice:
		for i := range providers {
			provider := &providers[i]
			if provider.Type == "oidc" && provider.DeviceLoginEnabled {
				return provider, authLoginModeDevice, nil
			}
		}
		return nil, "", fmt.Errorf("no OIDC provider with device login is enabled on server")
	case authLoginModeBuiltin:
		for i := range providers {
			provider := &providers[i]
			if provider.Type == "builtin" {
				return provider, authLoginModeBuiltin, nil
			}
		}
		return nil, "", fmt.Errorf("built-in auth is not enabled on server")
	default:
		if strings.EqualFold(requestedMode, "browser") {
			return nil, "", fmt.Errorf("browser auth mode is no longer supported; use --mode device or --mode builtin")
		}
		return nil, "", fmt.Errorf("unsupported auth mode %q", requestedMode)
	}
}

func fetchGatewayMode(ctx context.Context, baseURL string) (config.GatewayMode, bool) {
	var data struct {
		GatewayMode string `json:"gateway_mode"`
	}

	err := authRequest(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/metadata", "", nil, &data)
	if err != nil {
		return "", false
	}

	return config.ParseGatewayMode(data.GatewayMode)
}

func builtinLogin(ctx context.Context, baseURL, email, password string) (*authLoginData, error) {
	var data authLoginData
	err := authRequest(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/auth/login", "", map[string]string{
		"email":    email,
		"password": password,
	}, &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func refreshAccessToken(ctx context.Context, baseURL, refreshToken string) (*authLoginData, error) {
	var data authLoginData
	err := authRequest(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/auth/refresh", "", map[string]string{
		"refresh_token": refreshToken,
	}, &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func refreshProfileTeamGrants(ctx context.Context, cfg *config.Config, profileName string) (bool, error) {
	if cfg == nil {
		return false, nil
	}
	profile, err := cfg.GetProfile(profileName)
	if err != nil {
		return false, err
	}
	if strings.TrimSpace(*config.GetTokenVar()) != "" || strings.TrimSpace(os.Getenv(config.EnvToken)) != "" {
		return false, nil
	}
	refreshToken := strings.TrimSpace(profile.GetRefreshToken())
	if refreshToken == "" {
		return false, nil
	}

	refreshed, err := refreshAccessToken(ctx, profile.GetAPIURL(), refreshToken)
	if err != nil {
		return false, err
	}
	cfg.SetCredentials(
		profileName,
		profile.GetAPIURL(),
		refreshed.AccessToken,
		refreshed.RefreshToken,
		refreshed.ExpiresAt,
	)
	return true, nil
}

func printTeamGrantRefreshWarning(action string, refreshed bool, err error) {
	if err == nil && refreshed {
		return
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not refresh access token team grants after %s: %v\n", action, err)
	} else {
		fmt.Fprintf(os.Stderr, "Warning: access token team grants were not refreshed after %s.\n", action)
	}
	fmt.Fprintf(os.Stderr, "Hint: %s.\n", staleSelectedTeamGrantHint)
}

func withSelectedTeamAuthHint(err error) error {
	if err == nil || !isSelectedTeamAuthError(err) {
		return err
	}
	return fmt.Errorf("%w\nHint: %s", err, staleSelectedTeamGrantHint)
}

func isSelectedTeamAuthError(err error) bool {
	var apiErr *sandbox0.APIError
	if errors.As(err, &apiErr) {
		if apiErr.StatusCode != http.StatusUnauthorized {
			return false
		}
		return containsSelectedTeamAuthMessage(apiErr.Message) ||
			containsSelectedTeamAuthMessage(string(apiErr.Body)) ||
			containsSelectedTeamAuthMessage(err.Error())
	}
	return containsSelectedTeamAuthMessage(err.Error())
}

func containsSelectedTeamAuthMessage(value string) bool {
	return strings.Contains(strings.ToLower(value), "not a member of selected team")
}

func logoutToken(ctx context.Context, baseURL, accessToken string) error {
	return authRequest(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/auth/logout", accessToken, nil, nil)
}

func getProfileWithFreshToken() (*config.Profile, error) {
	cfg, err := getConfig()
	if err != nil {
		return nil, err
	}
	profileName := cfg.GetActiveProfile()
	p, err := cfg.GetProfile(profileName)
	if err != nil {
		return nil, err
	}

	if *config.GetTokenVar() != "" || os.Getenv(config.EnvToken) != "" {
		if p.GetToken() == "" {
			return nil, ErrNoToken
		}
		return p, nil
	}

	if p.GetToken() == "" {
		return nil, ErrNoToken
	}
	if p.ExpiresAt == 0 || p.GetRefreshToken() == "" {
		return p, nil
	}
	if time.Now().Unix() < p.ExpiresAt-30 {
		return p, nil
	}

	refreshed, err := refreshAccessToken(context.Background(), p.GetAPIURL(), p.GetRefreshToken())
	if err != nil {
		return nil, fmt.Errorf("token expired and refresh failed: %w (run `s0 auth login`)", err)
	}
	cfg.SetCredentials(
		profileName,
		p.GetAPIURL(),
		refreshed.AccessToken,
		refreshed.RefreshToken,
		refreshed.ExpiresAt,
	)
	if err := cfg.Save(); err != nil {
		return nil, fmt.Errorf("save refreshed credentials: %w", err)
	}
	updated, err := cfg.GetProfile(profileName)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func oidcLoginViaDeviceFlow(ctx context.Context, baseURL, providerID string) (*authLoginData, error) {
	type startResponse struct {
		DeviceLoginID           string `json:"device_login_id"`
		UserCode                string `json:"user_code"`
		VerificationURI         string `json:"verification_uri"`
		VerificationURIComplete string `json:"verification_uri_complete"`
		ExpiresAt               int64  `json:"expires_at"`
		IntervalSeconds         int    `json:"interval_seconds"`
	}
	var start startResponse
	if err := authRequest(
		ctx,
		http.MethodPost,
		strings.TrimRight(baseURL, "/")+"/auth/oidc/"+url.PathEscape(providerID)+"/device/start",
		"",
		nil,
		&start,
	); err != nil {
		return nil, err
	}

	fmt.Printf("Open %s and enter code %s to continue login.\n", start.VerificationURI, start.UserCode)
	if start.VerificationURIComplete != "" {
		if err := openBrowser(start.VerificationURIComplete); err != nil {
			fmt.Printf("Open browser failed, please open manually:\n%s\n", start.VerificationURIComplete)
		}
	}

	type pollResponse struct {
		Status          string         `json:"status"`
		IntervalSeconds int            `json:"interval_seconds"`
		ExpiresAt       int64          `json:"expires_at"`
		Login           *authLoginData `json:"login"`
	}

	interval := start.IntervalSeconds
	if interval <= 0 {
		interval = 5
	}

	for {
		if start.ExpiresAt > 0 && time.Now().Unix() >= start.ExpiresAt {
			return nil, fmt.Errorf("device login expired before completion")
		}

		var poll pollResponse
		if err := authRequest(
			ctx,
			http.MethodPost,
			strings.TrimRight(baseURL, "/")+"/auth/oidc/"+url.PathEscape(providerID)+"/device/poll",
			"",
			map[string]string{"device_login_id": start.DeviceLoginID},
			&poll,
		); err != nil {
			return nil, err
		}

		switch poll.Status {
		case "completed":
			if poll.Login == nil {
				return nil, fmt.Errorf("device login completed without login data")
			}
			return poll.Login, nil
		case "slow_down":
			if poll.IntervalSeconds > interval {
				interval = poll.IntervalSeconds
			} else {
				interval += 5
			}
		case "pending":
			if poll.IntervalSeconds > 0 {
				interval = poll.IntervalSeconds
			}
		default:
			return nil, fmt.Errorf("unexpected device login status %q", poll.Status)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(interval) * time.Second):
		}
	}
}

func maybeAutoSelectCurrentTeam(ctx context.Context, cfg *config.Config, profileName string) (*apispec.Team, bool, error) {
	if cfg == nil {
		return nil, false, nil
	}

	profile, err := cfg.GetProfile(profileName)
	if err != nil {
		return nil, false, err
	}
	if strings.TrimSpace(profile.GetToken()) == "" {
		return nil, false, nil
	}
	currentTeamID := strings.TrimSpace(profile.GetCurrentTeamID())

	client, err := newSDKClientForBaseURL(profile.GetAPIURL(), profile.GetToken())
	if err != nil {
		return nil, false, err
	}

	res, err := client.API().TeamsGet(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("list teams: %w", err)
	}
	successRes, ok := res.(*apispec.SuccessTeamListResponse)
	if !ok {
		return nil, false, fmt.Errorf("list teams: unexpected response type %T", res)
	}
	data, ok := successRes.Data.Get()
	if !ok {
		return nil, false, fmt.Errorf("list teams: missing response data")
	}

	for _, team := range data.Teams {
		if strings.TrimSpace(team.ID) == currentTeamID && currentTeamID != "" {
			return nil, false, nil
		}
	}

	if len(data.Teams) != 1 {
		if currentTeamID != "" {
			cfg.ClearCurrentTeam(profileName)
		}
		return nil, false, nil
	}

	team := data.Teams[0]
	homeRegionID, regionalGatewayURL, err := resolveCurrentTeamTarget(ctx, profile, client, team)
	if err != nil {
		return nil, false, err
	}
	cfg.SetCurrentTeam(profileName, team.ID, homeRegionID, regionalGatewayURL)
	return &team, true, nil
}

func shouldShowCurrentTeamSelectionHint(mode config.GatewayMode, currentTeamID string) bool {
	if mode != config.GatewayModeGlobal {
		return false
	}
	if strings.TrimSpace(currentTeamID) != "" {
		return false
	}
	return true
}

func openBrowser(targetURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", targetURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", targetURL)
	default:
		cmd = exec.Command("xdg-open", targetURL)
	}
	return cmd.Start()
}
