package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
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

// reviewStatus holds the review thread summary for a PR.
type reviewStatus struct {
	Unresolved    int
	Total         int
	ApproveCount  int
	ChangesCount  int
}

// combinedStatus returns a merged status string combining PR state and review status.
func combinedStatus(pr PR, rs reviewStatus) (label string, color string) {
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

// ignoredReviewers returns the set of reviewer logins to ignore from GV_IGNORE_REVIEWERS.
func ignoredReviewers() map[string]bool {
	env := os.Getenv("GV_IGNORE_REVIEWERS")
	if env == "" {
		return nil
	}
	m := make(map[string]bool)
	for _, name := range strings.Split(env, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			m[name] = true
		}
	}
	return m
}

// fetchReviewStatus returns unresolved/total review thread counts via GraphQL.
func fetchReviewStatus(pr PR, ignore map[string]bool) reviewStatus {
	fullRepo := pr.FullRepoName()
	parts := strings.SplitN(fullRepo, "/", 2)
	if len(parts) != 2 {
		return reviewStatus{}
	}
	query := `query($owner:String!,$repo:String!,$number:Int!){
  repository(owner:$owner,name:$repo){
    pullRequest(number:$number){
      reviewThreads(first:100){
        nodes{
          isResolved
          comments(first:1){
            nodes{ author{ login } }
          }
        }
      }
      latestOpinionatedReviews(first:100){
        nodes{
          state
          author{ login }
        }
      }
    }
  }
}`
	cmd := exec.Command("gh", "api", "graphql",
		"-f", fmt.Sprintf("query=%s", query),
		"-F", fmt.Sprintf("owner=%s", parts[0]),
		"-F", fmt.Sprintf("repo=%s", parts[1]),
		"-F", fmt.Sprintf("number=%d", pr.Number),
	)
	out, err := cmd.Output()
	if err != nil {
		return reviewStatus{}
	}
	var resp struct {
		Data struct {
			Repository struct {
				PullRequest struct {
					ReviewThreads struct {
						Nodes []struct {
							IsResolved bool `json:"isResolved"`
							Comments   struct {
								Nodes []struct {
									Author struct {
										Login string `json:"login"`
									} `json:"author"`
								} `json:"nodes"`
							} `json:"comments"`
						} `json:"nodes"`
					} `json:"reviewThreads"`
					LatestOpinionatedReviews struct {
						Nodes []struct {
							State  string `json:"state"`
							Author struct {
								Login string `json:"login"`
							} `json:"author"`
						} `json:"nodes"`
					} `json:"latestOpinionatedReviews"`
				} `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return reviewStatus{}
	}
	threads := resp.Data.Repository.PullRequest.ReviewThreads.Nodes
	total := 0
	unresolved := 0
	for _, t := range threads {
		if len(t.Comments.Nodes) > 0 && ignore[t.Comments.Nodes[0].Author.Login] {
			continue
		}
		total++
		if !t.IsResolved {
			unresolved++
		}
	}
	approveCount := 0
	changesCount := 0
	reviews := resp.Data.Repository.PullRequest.LatestOpinionatedReviews.Nodes
	for _, r := range reviews {
		if ignore[r.Author.Login] {
			continue
		}
		switch r.State {
		case "APPROVED":
			approveCount++
		case "CHANGES_REQUESTED":
			changesCount++
		}
	}
	return reviewStatus{Unresolved: unresolved, Total: total, ApproveCount: approveCount, ChangesCount: changesCount}
}

// fetchAllReviewStatuses fetches review status for all PRs in parallel.
func fetchAllReviewStatuses(prs []PR) map[string]reviewStatus {
	ignore := ignoredReviewers()
	result := make(map[string]reviewStatus)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, pr := range prs {
		wg.Add(1)
		go func(p PR) {
			defer wg.Done()
			rs := fetchReviewStatus(p, ignore)
			mu.Lock()
			result[p.HTMLURL] = rs
			mu.Unlock()
		}(pr)
	}
	wg.Wait()
	return result
}

type SearchResult struct {
	Items []PR `json:"items"`
}

func ghSearchPRs(qualifier string) ([]PR, error) {
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

func getUsername() (string, error) {
	cmd := exec.Command("gh", "api", "user", "-q", ".login")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get username: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

type fetchResult struct {
	prs []PR
	err error
}

func mergePRs(a, b []PR) []PR {
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

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}

func orgQualifier(org string) string {
	if org == "" {
		return ""
	}
	return fmt.Sprintf(" org:%s", org)
}

func splitEnv(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	for _, v := range strings.Split(s, ",") {
		v = strings.TrimSpace(v)
		if v != "" {
			result = append(result, v)
		}
	}
	return result
}

func formatAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func printHeader(username, org string) {
	header := fmt.Sprintf("\033[1;32m📋 GitHub Dashboard for @%s\033[0m", username)
	if org != "" {
		header += fmt.Sprintf(" \033[2m(org: %s)\033[0m", org)
	}
	fmt.Println(header)
}


func sortPRsByUpdated(prs []PR) {
	for i := 0; i < len(prs); i++ {
		for j := i + 1; j < len(prs); j++ {
			if prs[j].UpdatedAt.After(prs[i].UpdatedAt) {
				prs[i], prs[j] = prs[j], prs[i]
			}
		}
	}
}
