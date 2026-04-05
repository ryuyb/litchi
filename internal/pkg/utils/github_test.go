// Package utils provides common utility functions for the Litchi system.
package utils

import "testing"

func TestExtractOwner(t *testing.T) {
	tests := []struct {
		name     string
		repoName string
		expected string
	}{
		{"standard format", "owner/repo", "owner"},
		{"nested org", "org/team/repo", "org"},
		{"no slash", "repo", "repo"},
		{"empty string", "", ""},
		{"slash at end", "owner/", "owner"},
		{"slash at start", "/repo", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractOwner(tt.repoName)
			if result != tt.expected {
				t.Errorf("ExtractOwner(%s) = %s, expected %s", tt.repoName, result, tt.expected)
			}
		})
	}
}

func TestExtractRepo(t *testing.T) {
	tests := []struct {
		name     string
		repoName string
		expected string
	}{
		{"standard format", "owner/repo", "repo"},
		{"nested org", "org/team/repo", "team/repo"},
		{"no slash", "repo", "repo"},
		{"empty string", "", ""},
		{"slash at end", "owner/", ""},
		{"slash at start", "/repo", "repo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractRepo(tt.repoName)
			if result != tt.expected {
				t.Errorf("ExtractRepo(%s) = %s, expected %s", tt.repoName, result, tt.expected)
			}
		})
	}
}