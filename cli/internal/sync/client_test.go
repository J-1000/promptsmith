package sync

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name     string
		remote   string
		expected string
	}{
		{"default remote", "", DefaultRemote},
		{"custom remote", "https://custom.example.com", "https://custom.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.remote)
			if client.remote != tt.expected {
				t.Errorf("expected remote %s, got %s", tt.expected, client.remote)
			}
		})
	}
}

func TestSetToken(t *testing.T) {
	client := NewClient("")
	client.SetToken("test-token")
	if client.token != "test-token" {
		t.Errorf("expected token 'test-token', got %s", client.token)
	}
}

func TestSaveAndLoadToken(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "promptsmith-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	client := NewClient("")

	// Save token
	err = client.SaveToken(tmpDir, "my-secret-token")
	if err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	// Verify file permissions
	tokenPath := filepath.Join(tmpDir, TokenFileName)
	info, err := os.Stat(tokenPath)
	if err != nil {
		t.Fatalf("failed to stat token file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected token file permissions 0600, got %o", info.Mode().Perm())
	}

	// Load token
	err = client.LoadToken(tmpDir)
	if err != nil {
		t.Fatalf("failed to load token: %v", err)
	}
	if client.token != "my-secret-token" {
		t.Errorf("expected token 'my-secret-token', got %s", client.token)
	}
}

func TestLoadTokenFromEnv(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "promptsmith-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Set env var
	os.Setenv(TokenEnvVar, "env-token")
	defer os.Unsetenv(TokenEnvVar)

	client := NewClient("")
	err = client.LoadToken(tmpDir)
	if err != nil {
		t.Fatalf("failed to load token: %v", err)
	}
	if client.token != "env-token" {
		t.Errorf("expected token 'env-token', got %s", client.token)
	}
}

func TestLoadTokenNotLoggedIn(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "promptsmith-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	client := NewClient("")
	err = client.LoadToken(tmpDir)
	if err == nil {
		t.Error("expected error when not logged in")
	}
}

func TestDeleteToken(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "promptsmith-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	client := NewClient("")

	// Save and then delete
	err = client.SaveToken(tmpDir, "token-to-delete")
	if err != nil {
		t.Fatalf("failed to save token: %v", err)
	}

	err = client.DeleteToken(tmpDir)
	if err != nil {
		t.Fatalf("failed to delete token: %v", err)
	}

	// Verify file is gone
	tokenPath := filepath.Join(tmpDir, TokenFileName)
	if _, err := os.Stat(tokenPath); !os.IsNotExist(err) {
		t.Error("expected token file to be deleted")
	}
}

func TestLogin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/auth/login" {
			t.Errorf("expected /api/auth/login, got %s", r.URL.Path)
		}

		var req map[string]string
		json.NewDecoder(r.Body).Decode(&req)
		if req["email"] != "test@example.com" {
			t.Errorf("expected email 'test@example.com', got %s", req["email"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AuthResponse{
			Token:     "new-auth-token",
			ExpiresAt: time.Now().Add(24 * time.Hour),
			User: UserInfo{
				ID:    "user-123",
				Email: "test@example.com",
				Name:  "Test User",
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	auth, err := client.Login("test@example.com", "password123")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	if auth.Token != "new-auth-token" {
		t.Errorf("expected token 'new-auth-token', got %s", auth.Token)
	}
	if auth.User.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got %s", auth.User.Email)
	}
	if client.token != "new-auth-token" {
		t.Errorf("expected client token to be set")
	}
}

func TestLoginFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIError{
			Code:    "unauthorized",
			Message: "Invalid credentials",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.Login("test@example.com", "wrong-password")
	if err == nil {
		t.Error("expected error for failed login")
	}
}

func TestLoginWithToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/me" {
			t.Errorf("expected /api/auth/me, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer valid-token" {
			t.Errorf("expected auth header 'Bearer valid-token'")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(UserInfo{
			ID:    "user-123",
			Email: "token@example.com",
			Name:  "Token User",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	user, err := client.LoginWithToken("valid-token")
	if err != nil {
		t.Fatalf("login with token failed: %v", err)
	}
	if user.Email != "token@example.com" {
		t.Errorf("expected email 'token@example.com', got %s", user.Email)
	}
	if client.token != "valid-token" {
		t.Error("expected client token to be set")
	}
}

func TestLoginWithTokenFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIError{
			Code:    "unauthorized",
			Message: "Invalid token",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.LoginWithToken("invalid-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestLogout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/auth/logout" {
			t.Errorf("expected /api/auth/logout, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetToken("test-token")
	err := client.Logout()
	if err != nil {
		t.Fatalf("logout failed: %v", err)
	}
	if client.token != "" {
		t.Error("expected client token to be cleared")
	}
}

func TestLogoutFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetToken("test-token")
	err := client.Logout()
	if err == nil {
		t.Error("expected error for failed logout")
	}
}

func TestPush(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/sync/push" {
			t.Errorf("expected /api/sync/push, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected auth header 'Bearer test-token'")
		}

		var req PushRequest
		json.NewDecoder(r.Body).Decode(&req)
		if len(req.Prompts) != 1 {
			t.Errorf("expected 1 prompt, got %d", len(req.Prompts))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PushResponse{
			Synced:  1,
			Message: "Pushed successfully",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetToken("test-token")

	req := &PushRequest{
		Project: Project{
			ID:   "proj-123",
			Name: "Test Project",
		},
		Prompts: []Prompt{
			{ID: "prompt-1", Name: "test-prompt"},
		},
	}

	resp, err := client.Push(req)
	if err != nil {
		t.Fatalf("push failed: %v", err)
	}
	if resp.Synced != 1 {
		t.Errorf("expected 1 synced, got %d", resp.Synced)
	}
}

func TestPull(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/sync/pull/proj-123" {
			t.Errorf("expected /api/sync/pull/proj-123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PullResponse{
			Project: Project{
				ID:   "proj-123",
				Name: "Test Project",
			},
			Prompts: []Prompt{
				{ID: "prompt-1", Name: "pulled-prompt"},
			},
			Message: "Pulled successfully",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetToken("test-token")

	resp, err := client.Pull("proj-123", nil)
	if err != nil {
		t.Fatalf("pull failed: %v", err)
	}
	if len(resp.Prompts) != 1 {
		t.Errorf("expected 1 prompt, got %d", len(resp.Prompts))
	}
	if resp.Prompts[0].Name != "pulled-prompt" {
		t.Errorf("expected prompt name 'pulled-prompt', got %s", resp.Prompts[0].Name)
	}
}

func TestWhoAmI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/me" {
			t.Errorf("expected /api/auth/me, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(UserInfo{
			ID:    "user-123",
			Email: "test@example.com",
			Name:  "Test User",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetToken("test-token")

	user, err := client.WhoAmI()
	if err != nil {
		t.Fatalf("whoami failed: %v", err)
	}
	if user.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got %s", user.Email)
	}
}

func TestGetProject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/projects/proj-123" {
			t.Errorf("expected /api/projects/proj-123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Project{
			ID:   "proj-123",
			Name: "Test Project",
			Team: "my-team",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetToken("test-token")

	project, err := client.GetProject("proj-123")
	if err != nil {
		t.Fatalf("get project failed: %v", err)
	}
	if project.Name != "Test Project" {
		t.Errorf("expected name 'Test Project', got %s", project.Name)
	}
}

func TestGetProjectNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetToken("test-token")

	project, err := client.GetProject("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if project != nil {
		t.Error("expected nil project for not found")
	}
}

func TestCreateProject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/projects" {
			t.Errorf("expected /api/projects, got %s", r.URL.Path)
		}

		var req map[string]string
		json.NewDecoder(r.Body).Decode(&req)
		if req["name"] != "New Project" {
			t.Errorf("expected name 'New Project', got %s", req["name"])
		}

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Project{
			ID:   "new-proj-id",
			Name: "New Project",
			Team: "my-team",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetToken("test-token")

	project, err := client.CreateProject("New Project", "my-team")
	if err != nil {
		t.Fatalf("create project failed: %v", err)
	}
	if project.ID != "new-proj-id" {
		t.Errorf("expected ID 'new-proj-id', got %s", project.ID)
	}
}

func TestAPIError(t *testing.T) {
	err := &APIError{
		Code:    "validation_error",
		Message: "Invalid request",
	}
	expected := "validation_error: Invalid request"
	if err.Error() != expected {
		t.Errorf("expected error string '%s', got '%s'", expected, err.Error())
	}
}
