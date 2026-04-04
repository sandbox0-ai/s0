package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/sandbox0-ai/s0/internal/config"
)

type authProvider struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Type                string `json:"type"`
	BrowserLoginEnabled bool   `json:"browser_login_enabled"`
	DeviceLoginEnabled  bool   `json:"device_login_enabled"`
}

type authLoginData struct {
	AccessToken     string `json:"access_token"`
	RefreshToken    string `json:"refresh_token"`
	ExpiresAt       int64  `json:"expires_at"`
	RegionalSession *struct {
		RegionID           string `json:"region_id"`
		RegionalGatewayURL string `json:"regional_gateway_url"`
		Token              string `json:"token"`
		ExpiresAt          int64  `json:"expires_at"`
	} `json:"regional_session,omitempty"`
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
	authLoginModeBrowser authLoginMode = "browser"
	authLoginModeBuiltin authLoginMode = "builtin"
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
			if provider.Type == "oidc" && provider.BrowserLoginEnabled {
				return provider, authLoginModeBrowser, nil
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
	case authLoginModeBrowser:
		for i := range providers {
			provider := &providers[i]
			if provider.Type == "oidc" && provider.BrowserLoginEnabled {
				return provider, authLoginModeBrowser, nil
			}
		}
		return nil, "", fmt.Errorf("no OIDC provider with browser login is enabled on server")
	case authLoginModeBuiltin:
		for i := range providers {
			provider := &providers[i]
			if provider.Type == "builtin" {
				return provider, authLoginModeBuiltin, nil
			}
		}
		return nil, "", fmt.Errorf("built-in auth is not enabled on server")
	default:
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
		toRegionalSessionConfig(refreshed),
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

func oidcLoginViaBrowser(ctx context.Context, baseURL, providerID string) (*authLoginData, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen callback: %w", err)
	}
	defer func() {
		_ = ln.Close()
	}()

	resultCh := make(chan *authLoginData, 1)
	errCh := make(chan error, 1)
	mux := http.NewServeMux()
	server := &http.Server{Handler: mux}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if errMsg := q.Get("error"); errMsg != "" {
			http.Error(w, errMsg, http.StatusBadRequest)
			select {
			case errCh <- fmt.Errorf("%s", errMsg):
			default:
			}
			return
		}

		expiresUnix, err := strconv.ParseInt(q.Get("expires_unix"), 10, 64)
		if err != nil {
			http.Error(w, "missing or invalid expires_unix", http.StatusBadRequest)
			select {
			case errCh <- fmt.Errorf("missing or invalid expires_unix"):
			default:
			}
			return
		}

		data := &authLoginData{
			AccessToken:  q.Get("access_token"),
			RefreshToken: q.Get("refresh_token"),
			ExpiresAt:    expiresUnix,
		}
		regionalExpiresUnix, err := strconv.ParseInt(q.Get("regional_expires_unix"), 10, 64)
		if err == nil &&
			q.Get("regional_access_token") != "" &&
			q.Get("regional_gateway_url") != "" &&
			q.Get("region_id") != "" {
			data.RegionalSession = &struct {
				RegionID           string `json:"region_id"`
				RegionalGatewayURL string `json:"regional_gateway_url"`
				Token              string `json:"token"`
				ExpiresAt          int64  `json:"expires_at"`
			}{
				RegionID:           q.Get("region_id"),
				RegionalGatewayURL: q.Get("regional_gateway_url"),
				Token:              q.Get("regional_access_token"),
				ExpiresAt:          regionalExpiresUnix,
			}
		}
		if data.AccessToken == "" || data.RefreshToken == "" {
			http.Error(w, "missing tokens in callback", http.StatusBadRequest)
			select {
			case errCh <- fmt.Errorf("missing tokens in callback"):
			default:
			}
			return
		}

		_, _ = w.Write([]byte("Sandbox0 CLI login successful. You can close this window."))
		select {
		case resultCh <- data:
		default:
		}
	})

	go func() {
		if serveErr := server.Serve(ln); serveErr != nil && serveErr != http.ErrServerClosed {
			select {
			case errCh <- serveErr:
			default:
			}
		}
	}()

	returnURL := (&url.URL{
		Scheme: "http",
		Host:   ln.Addr().String(),
		Path:   "/callback",
	}).String()
	loginURL := buildOIDCLoginURL(baseURL, providerID, returnURL)

	fmt.Printf("Opening browser for %s login...\n", providerID)
	if err := openBrowser(loginURL); err != nil {
		fmt.Printf("Open browser failed, please open manually:\n%s\n", loginURL)
	}

	select {
	case <-ctx.Done():
		_ = server.Shutdown(context.Background())
		return nil, ctx.Err()
	case err := <-errCh:
		_ = server.Shutdown(context.Background())
		return nil, err
	case data := <-resultCh:
		_ = server.Shutdown(context.Background())
		return data, nil
	case <-time.After(3 * time.Minute):
		_ = server.Shutdown(context.Background())
		return nil, fmt.Errorf("oidc login timed out")
	}
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

func buildOIDCLoginURL(baseURL, providerID, returnURL string) string {
	return fmt.Sprintf("%s/auth/oidc/%s/login?return_url=%s",
		strings.TrimRight(baseURL, "/"),
		url.PathEscape(providerID),
		url.QueryEscape(returnURL),
	)
}

func shouldShowFirstTeamOnboardingHint(ctx context.Context, baseURL string, data *authLoginData) bool {
	if data == nil || data.RegionalSession != nil {
		return false
	}

	mode, ok := fetchGatewayMode(ctx, baseURL)
	return ok && mode == config.GatewayModeGlobal
}

func toRegionalSessionConfig(data *authLoginData) *config.RegionalSession {
	if data == nil || data.RegionalSession == nil {
		return nil
	}
	if data.RegionalSession.Token == "" ||
		data.RegionalSession.RegionalGatewayURL == "" ||
		data.RegionalSession.RegionID == "" ||
		data.RegionalSession.ExpiresAt == 0 {
		return nil
	}
	return &config.RegionalSession{
		Token:      data.RegionalSession.Token,
		GatewayURL: data.RegionalSession.RegionalGatewayURL,
		RegionID:   data.RegionalSession.RegionID,
		ExpiresAt:  data.RegionalSession.ExpiresAt,
	}
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
