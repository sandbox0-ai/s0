package skills

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Release struct {
	Name           string   `json:"name"`
	ReleaseVersion string   `json:"releaseVersion"`
	ReleaseTag     string   `json:"releaseTag"`
	ArtifactPrefix string   `json:"artifactPrefix"`
	SourcePriority []string `json:"sourcePriority"`
	DownloadURL    string   `json:"downloadUrl"`
	ChecksumURL    string   `json:"checksumUrl"`
	ManifestURL    string   `json:"manifestUrl"`
}

type apiEnvelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type APIClient struct {
	baseURL    string
	token      string
	userAgent  string
	httpClient *http.Client
}

func NewAPIClient(baseURL, token, userAgent string) *APIClient {
	return &APIClient{
		baseURL:   strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		token:     strings.TrimSpace(token),
		userAgent: strings.TrimSpace(userAgent),
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

func (c *APIClient) GetRelease(ctx context.Context, name string) (*Release, error) {
	endpoint := c.baseURL + "/api/v1/agent-skills/" + name
	req, err := c.newRequest(ctx, http.MethodGet, endpoint)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request agent skill metadata: %w", err)
	}
	defer resp.Body.Close()

	var envelope apiEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("decode agent skill metadata: %w", err)
	}
	if !envelope.Success {
		if envelope.Error != nil && envelope.Error.Message != "" {
			return nil, fmt.Errorf("%s", envelope.Error.Message)
		}
		return nil, fmt.Errorf("agent skill metadata request failed with status %d", resp.StatusCode)
	}

	var release Release
	if err := json.Unmarshal(envelope.Data, &release); err != nil {
		return nil, fmt.Errorf("decode agent skill data: %w", err)
	}
	return &release, nil
}

func (c *APIClient) DownloadArtifact(ctx context.Context, name string, dest io.Writer) (string, error) {
	endpoint := c.baseURL + "/api/v1/agent-skills/" + name + "/download"
	return c.download(ctx, endpoint, dest)
}

func (c *APIClient) DownloadChecksum(ctx context.Context, name string) (string, error) {
	endpoint := c.baseURL + "/api/v1/agent-skills/" + name + "/checksum"
	req, err := c.newRequest(ctx, http.MethodGet, endpoint)
	if err != nil {
		return "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request checksum: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("checksum request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read checksum: %w", err)
	}
	fields := strings.Fields(string(body))
	if len(fields) == 0 {
		return "", fmt.Errorf("checksum response is empty")
	}
	return fields[0], nil
}

func SHA256File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("hash %s: %w", path, err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (c *APIClient) download(ctx context.Context, endpoint string, dest io.Writer) (string, error) {
	req, err := c.newRequest(ctx, http.MethodGet, endpoint)
	if err != nil {
		return "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request artifact: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("artifact request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if _, err := io.Copy(dest, resp.Body); err != nil {
		return "", fmt.Errorf("write artifact: %w", err)
	}
	return resp.Request.URL.String(), nil
}

func (c *APIClient) newRequest(ctx context.Context, method, endpoint string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	return req, nil
}
