package github

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// SearchPRs searches GitHub for pull requests matching the given qualifier.
func SearchPRs(qualifier string) ([]PR, error) {
	cmd := exec.Command("gh", "api", "search/issues",
		"-X", "GET",
		"-f", fmt.Sprintf("q=%s type:pr state:open", qualifier),
		"-f", "per_page=100",
		"-f", "sort=updated",
		"-f", "order=desc",
	)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh api failed: %s", string(exitErr.Stderr))
		}
		return nil, err
	}
	var result SearchResult
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("json parse error: %w", err)
	}
	return result.Items, nil
}

// GetUsername returns the authenticated GitHub username.
func GetUsername() (string, error) {
	cmd := exec.Command("gh", "api", "user", "-q", ".login")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get username: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// MergePRs merges two PR slices, deduplicating by HTMLURL.
func MergePRs(a, b []PR) []PR {
	seen := make(map[string]bool)
	var merged []PR
	for _, pr := range a {
		if !seen[pr.HTMLURL] {
			seen[pr.HTMLURL] = true
			merged = append(merged, pr)
		}
	}
	for _, pr := range b {
		if !seen[pr.HTMLURL] {
			seen[pr.HTMLURL] = true
			merged = append(merged, pr)
		}
	}
	return merged
}

// SortPRsByUpdated sorts PRs by updated_at in descending order.
func SortPRsByUpdated(prs []PR) {
	for i := 0; i < len(prs); i++ {
		for j := i + 1; j < len(prs); j++ {
			if prs[j].UpdatedAt.After(prs[i].UpdatedAt) {
				prs[i], prs[j] = prs[j], prs[i]
			}
		}
	}
}
