package scanner

import (
	"regexp"
	"strings"
)

type Secret struct {
	Type    string
	Line    int
	Match   string
	Context string
}

type Scanner struct {
	patterns []secretPattern
}

type secretPattern struct {
	name    string
	pattern *regexp.Regexp
}

func New() *Scanner {
	return &Scanner{
		patterns: []secretPattern{
			{"AWS Access Key", regexp.MustCompile(`(?i)(AKIA[0-9A-Z]{16})`)},
			{"AWS Secret Key", regexp.MustCompile(`(?i)aws(.{0,20})?['\"][0-9a-zA-Z/+]{40}['\"]`)},
			{"GitHub Token", regexp.MustCompile(`(?i)(ghp_[a-zA-Z0-9]{36}|github_pat_[a-zA-Z0-9]{22}_[a-zA-Z0-9]{59})`)},
			{"GitLab Token", regexp.MustCompile(`(?i)(glpat-[a-zA-Z0-9\-]{20})`)},
			{"OpenAI API Key", regexp.MustCompile(`(?i)(sk-[a-zA-Z0-9]{48})`)},
			{"Anthropic API Key", regexp.MustCompile(`(?i)(sk-ant-[a-zA-Z0-9\-]{95})`)},
			{"Google API Key", regexp.MustCompile(`(?i)(AIza[0-9A-Za-z\-_]{35})`)},
			{"Slack Token", regexp.MustCompile(`(?i)(xox[baprs]-[0-9]{10,13}-[0-9]{10,13}[a-zA-Z0-9-]*)`)},
			{"Stripe API Key", regexp.MustCompile(`(?i)(sk_live_[0-9a-zA-Z]{24})`)},
			{"Private Key", regexp.MustCompile(`(?i)-----BEGIN (RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`)},
			{"Generic Secret", regexp.MustCompile(`(?i)(api[_-]?key|secret[_-]?key|password|passwd|pwd|token)['\"]?\s*[:=]\s*['\"][a-zA-Z0-9+/=]{16,}['\"]`)},
			{"Bearer Token", regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9\-_.~+/]+=*`)},
			{"Basic Auth", regexp.MustCompile(`(?i)basic\s+[a-zA-Z0-9+/]+=*`)},
			{"Database URL", regexp.MustCompile(`(?i)(postgres|mysql|mongodb|redis)://[^@]+@[^\s]+`)},
		},
	}
}

func (s *Scanner) Scan(content string) []Secret {
	var secrets []Secret
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		for _, pattern := range s.patterns {
			if matches := pattern.pattern.FindAllString(line, -1); len(matches) > 0 {
				for _, match := range matches {
					secrets = append(secrets, Secret{
						Type:    pattern.name,
						Line:    lineNum + 1,
						Match:   maskSecret(match),
						Context: truncateLine(line, 80),
					})
				}
			}
		}
	}

	return secrets
}

func maskSecret(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "..." + s[len(s)-4:]
}

func truncateLine(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
