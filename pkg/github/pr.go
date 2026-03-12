package github

import (
	"fmt"
	"strings"
	"time"
)

// PR represents a GitHub pull request from the search API.
type PR struct {
	RepositoryURL string    `json:"repository_url"`
	Number        int       `json:"number"`
	Title         string    `json:"title"`
	HTMLURL       string    `json:"html_url"`
	UpdatedAt     time.Time `json:"updated_at"`
	Draft         bool      `json:"draft"`
	User          struct {
		Login string `json:"login"`
		Type  string `json:"type"`
	} `json:"user"`
}

func (pr PR) FullRepoName() string {
	const prefix = "https://api.github.com/repos/"
	return strings.TrimPrefix(pr.RepositoryURL, prefix)
}

func (pr PR) RepoName() string {
	full := pr.FullRepoName()
	if i := strings.Index(full, "/"); i >= 0 {
		return full[i+1:]
	}
	return full
}

// SearchResult holds the response from GitHub search API.
type SearchResult struct {
	Items []PR `json:"items"`
}

// ReviewStatus holds the review thread summary for a PR.
type ReviewStatus struct {
	Unresolved   int
	Total        int
	ApproveCount int
	ChangesCount int
}

// FetchResult holds the result of a PR fetch operation.
type FetchResult struct {
	PRs []PR
	Err error
}

// CombinedStatus returns a merged status string combining PR state and review status.
func CombinedStatus(pr PR, rs ReviewStatus) (label string, color string) {
	if pr.Draft {
		return "draft", "\033[2m"
	}
	if rs.ChangesCount > 0 {
		suffix := ""
		if rs.Unresolved > 0 {
			suffix = fmt.Sprintf(", %d unresolved", rs.Unresolved)
		}
		return fmt.Sprintf("changes requested (%d)%s", rs.ChangesCount, suffix), "\033[31m"
	}
	if rs.ApproveCount > 0 {
		suffix := ""
		if rs.Unresolved > 0 {
			suffix = fmt.Sprintf(", %d unresolved", rs.Unresolved)
		}
		return fmt.Sprintf("approved (%d)%s", rs.ApproveCount, suffix), "\033[1;32m"
	}
	if rs.Unresolved > 0 {
		return fmt.Sprintf("reviewed (%d unresolved)", rs.Unresolved), "\033[33m"
	}
	return "open", "\033[32m"
}
