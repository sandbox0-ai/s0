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
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
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
	cfg.SetCredentials(profileName, p.GetAPIURL(), refreshed.AccessToken, refreshed.RefreshToken, refreshed.ExpiresAt)
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
	loginURL := fmt.Sprintf("%s/auth/oidc/%s/login?return_url=%s",
		strings.TrimRight(baseURL, "/"),
		url.PathEscape(providerID),
		url.QueryEscape(returnURL),
	)

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
