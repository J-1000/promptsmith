package sync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	DefaultRemote   = "https://api.promptsmith.dev"
	TokenFileName   = "token"
	TokenEnvVar     = "PROMPTSMITH_TOKEN"
)

type Client struct {
	remote     string
	token      string
	httpClient *http.Client
}

type AuthResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      UserInfo  `json:"user"`
}

type UserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Team        string    `json:"team,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Prompt struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	FilePath    string    `json:"file_path"`
	CreatedAt   time.Time `json:"created_at"`
}

type PromptVersion struct {
	ID              string    `json:"id"`
	PromptID        string    `json:"prompt_id"`
	Version         string    `json:"version"`
	Content         string    `json:"content"`
	Variables       string    `json:"variables"`
	Metadata        string    `json:"metadata"`
	ParentVersionID *string   `json:"parent_version_id,omitempty"`
	CommitMessage   string    `json:"commit_message"`
	CreatedAt       time.Time `json:"created_at"`
	CreatedBy       string    `json:"created_by"`
}

type Tag struct {
	ID        string    `json:"id"`
	PromptID  string    `json:"prompt_id"`
	VersionID string    `json:"version_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type PushRequest struct {
	Project  Project         `json:"project"`
	Prompts  []Prompt        `json:"prompts"`
	Versions []PromptVersion `json:"versions"`
	Tags     []Tag           `json:"tags"`
}

type PushResponse struct {
	Synced    int      `json:"synced"`
	Conflicts []string `json:"conflicts,omitempty"`
	Message   string   `json:"message"`
}

type PullResponse struct {
	Project  Project         `json:"project"`
	Prompts  []Prompt        `json:"prompts"`
	Versions []PromptVersion `json:"versions"`
	Tags     []Tag           `json:"tags"`
	Message  string          `json:"message"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func NewClient(remote string) *Client {
	if remote == "" {
		remote = DefaultRemote
	}
	return &Client{
		remote: remote,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) SetToken(token string) {
	c.token = token
}

func (c *Client) LoadToken(configDir string) error {
	// First check env var
	if token := os.Getenv(TokenEnvVar); token != "" {
		c.token = token
		return nil
	}

	// Then check token file
	tokenPath := filepath.Join(configDir, TokenFileName)
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("not logged in. Run 'promptsmith login' first")
		}
		return fmt.Errorf("failed to read token: %w", err)
	}
	c.token = string(bytes.TrimSpace(data))
	return nil
}

func (c *Client) SaveToken(configDir, token string) error {
	tokenPath := filepath.Join(configDir, TokenFileName)
	if err := os.WriteFile(tokenPath, []byte(token), 0600); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}
	return nil
}

func (c *Client) DeleteToken(configDir string) error {
	tokenPath := filepath.Join(configDir, TokenFileName)
	if err := os.Remove(tokenPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete token: %w", err)
	}
	return nil
}

func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.remote+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	return c.httpClient.Do(req)
}

func (c *Client) Login(email, password string) (*AuthResponse, error) {
	resp, err := c.doRequest("POST", "/api/auth/login", map[string]string{
		"email":    email,
		"password": password,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			return nil, fmt.Errorf("login failed with status %d", resp.StatusCode)
		}
		return nil, &apiErr
	}

	var auth AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&auth); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.token = auth.Token
	return &auth, nil
}

func (c *Client) LoginWithToken(token string) (*UserInfo, error) {
	c.token = token
	resp, err := c.doRequest("GET", "/api/auth/me", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			return nil, fmt.Errorf("token validation failed with status %d", resp.StatusCode)
		}
		return nil, &apiErr
	}

	var user UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &user, nil
}

func (c *Client) Logout() error {
	resp, err := c.doRequest("POST", "/api/auth/logout", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("logout failed with status %d", resp.StatusCode)
	}

	c.token = ""
	return nil
}

func (c *Client) Push(req *PushRequest) (*PushResponse, error) {
	resp, err := c.doRequest("POST", "/api/sync/push", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			return nil, fmt.Errorf("push failed with status %d", resp.StatusCode)
		}
		return nil, &apiErr
	}

	var pushResp PushResponse
	if err := json.NewDecoder(resp.Body).Decode(&pushResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &pushResp, nil
}

func (c *Client) Pull(projectID string, since *time.Time) (*PullResponse, error) {
	path := fmt.Sprintf("/api/sync/pull/%s", projectID)
	if since != nil {
		path += fmt.Sprintf("?since=%s", since.Format(time.RFC3339))
	}

	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			return nil, fmt.Errorf("pull failed with status %d", resp.StatusCode)
		}
		return nil, &apiErr
	}

	var pullResp PullResponse
	if err := json.NewDecoder(resp.Body).Decode(&pullResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &pullResp, nil
}

func (c *Client) GetProject(projectID string) (*Project, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/projects/%s", projectID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			return nil, fmt.Errorf("get project failed with status %d", resp.StatusCode)
		}
		return nil, &apiErr
	}

	var project Project
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &project, nil
}

func (c *Client) CreateProject(name, team string) (*Project, error) {
	resp, err := c.doRequest("POST", "/api/projects", map[string]string{
		"name": name,
		"team": team,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var apiErr APIError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			return nil, fmt.Errorf("create project failed with status %d", resp.StatusCode)
		}
		return nil, &apiErr
	}

	var project Project
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &project, nil
}

func (c *Client) WhoAmI() (*UserInfo, error) {
	resp, err := c.doRequest("GET", "/api/auth/me", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			return nil, fmt.Errorf("whoami failed with status %d", resp.StatusCode)
		}
		return nil, &apiErr
	}

	var user UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &user, nil
}
