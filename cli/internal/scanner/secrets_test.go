package scanner

import (
	"testing"
)

func TestScannerDetectsSecrets(t *testing.T) {
	s := New()

	tests := []struct {
		name        string
		content     string
		expectType  string
		shouldFind  bool
	}{
		{
			name:        "AWS Access Key",
			content:     "aws_access_key = AKIAIOSFODNN7EXAMPLE",
			expectType:  "AWS Access Key",
			shouldFind:  true,
		},
		{
			name:        "GitHub Token",
			content:     "token: ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			expectType:  "GitHub Token",
			shouldFind:  true,
		},
		{
			name:        "OpenAI API Key",
			content:     "OPENAI_API_KEY=sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			expectType:  "OpenAI API Key",
			shouldFind:  true,
		},
		{
			name:        "Private Key Header",
			content:     "-----BEGIN RSA PRIVATE KEY-----",
			expectType:  "Private Key",
			shouldFind:  true,
		},
		{
			name:        "Generic API Key",
			content:     `api_key = "abcdefghijklmnopqrstuvwxyz123456"`,
			expectType:  "Generic Secret",
			shouldFind:  true,
		},
		{
			name:        "Database URL",
			content:     "DATABASE_URL=postgres://user:password@localhost:5432/mydb",
			expectType:  "Database URL",
			shouldFind:  true,
		},
		{
			name:        "Bearer Token",
			content:     "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expectType:  "Bearer Token",
			shouldFind:  true,
		},
		{
			name:        "No secrets",
			content:     "This is just plain text without any secrets",
			expectType:  "",
			shouldFind:  false,
		},
		{
			name:        "Mustache variable (not a secret)",
			content:     "Hello {{name}}, your API key is {{api_key}}",
			expectType:  "",
			shouldFind:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secrets := s.Scan(tt.content)

			if tt.shouldFind {
				if len(secrets) == 0 {
					t.Errorf("expected to find secret of type '%s', but found none", tt.expectType)
					return
				}

				found := false
				for _, secret := range secrets {
					if secret.Type == tt.expectType {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected to find secret of type '%s', found types: %v", tt.expectType, secrets)
				}
			} else {
				if len(secrets) > 0 {
					t.Errorf("expected no secrets, but found: %v", secrets)
				}
			}
		})
	}
}

func TestScannerReturnsCorrectLineNumbers(t *testing.T) {
	s := New()

	content := `line 1: nothing here
line 2: still nothing
line 3: AKIAIOSFODNN7EXAMPLE
line 4: normal text`

	secrets := s.Scan(content)

	if len(secrets) != 1 {
		t.Fatalf("expected 1 secret, got %d", len(secrets))
	}

	if secrets[0].Line != 3 {
		t.Errorf("expected line 3, got %d", secrets[0].Line)
	}
}

func TestMaskSecret(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"short", "****"},
		{"12345678", "****"},
		{"123456789", "1234...6789"},
		{"abcdefghijklmnop", "abcd...mnop"},
	}

	for _, tt := range tests {
		result := maskSecret(tt.input)
		if result != tt.expected {
			t.Errorf("maskSecret(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestTruncateLine(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 80, "short"},
		{"  trimmed  ", 80, "trimmed"},
		{"this is a very long line that should be truncated", 20, "this is a very lo..."},
	}

	for _, tt := range tests {
		result := truncateLine(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateLine(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestScannerMultipleSecretsPerLine(t *testing.T) {
	s := New()

	// Line with multiple secrets
	content := "AKIAIOSFODNN7EXAMPLE and also ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

	secrets := s.Scan(content)

	if len(secrets) < 2 {
		t.Errorf("expected at least 2 secrets, got %d", len(secrets))
	}
}

func TestScannerMultilineContent(t *testing.T) {
	s := New()

	content := `# Configuration file
DATABASE_URL=postgres://admin:secret123@db.example.com:5432/prod
API_KEY="sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
DEBUG=true
AKIAIOSFODNN7EXAMPLE
`

	secrets := s.Scan(content)

	// Should find: Database URL, OpenAI key, AWS key
	if len(secrets) < 3 {
		t.Errorf("expected at least 3 secrets, got %d: %v", len(secrets), secrets)
	}

	// Verify line numbers are correct
	lineNumbers := make(map[int]bool)
	for _, s := range secrets {
		lineNumbers[s.Line] = true
	}

	if !lineNumbers[2] {
		t.Error("expected to find secret on line 2 (DATABASE_URL)")
	}
	if !lineNumbers[3] {
		t.Error("expected to find secret on line 3 (API_KEY)")
	}
	if !lineNumbers[5] {
		t.Error("expected to find secret on line 5 (AWS key)")
	}
}
