package github

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// ParseIgnoredReviewers parses a comma-separated list of reviewer logins to ignore.
func ParseIgnoredReviewers(csv string) map[string]bool {
	if csv == "" {
		return nil
	}
	m := make(map[string]bool)
	for _, name := range strings.Split(csv, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			m[name] = true
		}
	}
	return m
}

// FetchReviewStatus returns unresolved/total review thread counts via GraphQL.
func FetchReviewStatus(pr PR, ignore map[string]bool) ReviewStatus {
	fullRepo := pr.FullRepoName()
	parts := strings.SplitN(fullRepo, "/", 2)
	if len(parts) != 2 {
		return ReviewStatus{}
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
		return ReviewStatus{}
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
		return ReviewStatus{}
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
	return ReviewStatus{Unresolved: unresolved, Total: total, ApproveCount: approveCount, ChangesCount: changesCount}
}

// FetchAllReviewStatuses fetches review status for all PRs in parallel.
func FetchAllReviewStatuses(prs []PR, ignore map[string]bool) map[string]ReviewStatus {
	result := make(map[string]ReviewStatus)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, pr := range prs {
		wg.Add(1)
		go func(p PR) {
			defer wg.Done()
			rs := FetchReviewStatus(p, ignore)
			mu.Lock()
			result[p.HTMLURL] = rs
			mu.Unlock()
		}(pr)
	}
	wg.Wait()
	return result
}
