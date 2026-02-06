package git

import (
	"testing"
)

func TestConvertSSHToHTTPS(t *testing.T) {
	tests := []struct {
		name     string
		sshURL   string
		expected string
	}{
		{
			name:     "GitHub SSH URL",
			sshURL:   "git@github.com:muidea/skills-repo.git",
			expected: "https://github.com/muidea/skills-repo",
		},
		{
			name:     "GitLab SSH URL",
			sshURL:   "git@gitlab.com:group/project.git",
			expected: "https://gitlab.com/group/project",
		},
		{
			name:     "SSH protocol URL",
			sshURL:   "ssh://git@github.com/user/repo.git",
			expected: "https://github.com/user/repo",
		},
		{
			name:     "Invalid URL",
			sshURL:   "not-a-valid-url",
			expected: "",
		},
		{
			name:     "HTTPS URL should return empty",
			sshURL:   "https://github.com/user/repo.git",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertSSHToHTTPS(tt.sshURL)
			if result != tt.expected {
				t.Errorf("convertSSHToHTTPS(%q) = %q, want %q", tt.sshURL, result, tt.expected)
			}
		})
	}
}

func TestGetSSHAuth(t *testing.T) {
	// This is a basic test to ensure the function doesn't panic
	repo := &Repository{}
	_, err := repo.getSSHAuth()

	// We expect an error because we're not in a real environment with SSH keys
	// But the function should not panic
	if err == nil {
		t.Log("getSSHAuth returned no error (might be in CI environment with SSH agent)")
	} else {
		t.Logf("getSSHAuth returned expected error: %v", err)
	}
}
