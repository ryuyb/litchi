// Package utils provides common utility functions for the Litchi system.
package utils

// ExtractOwner extracts the owner from owner/repo format.
// If no slash is found, returns the full string as owner.
func ExtractOwner(repoName string) string {
	for i := 0; i < len(repoName); i++ {
		if repoName[i] == '/' {
			return repoName[:i]
		}
	}
	return repoName
}

// ExtractRepo extracts the repo name from owner/repo format.
// If no slash is found, returns the full string as repo.
func ExtractRepo(repoName string) string {
	for i := 0; i < len(repoName); i++ {
		if repoName[i] == '/' {
			return repoName[i+1:]
		}
	}
	return repoName
}