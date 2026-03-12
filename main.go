package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type PR struct {
	RepositoryURL string    `json:"repository_url"`
	Number        int       `json:"number"`
	Title         string    `json:"title"`
	HTMLURL       string    `json:"html_url"`
	UpdatedAt     time.Time `json:"updated_at"`
	Draft         bool      `json:"draft"`
	User struct {
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
	Unresolved int
	Total      int
}

// combinedStatus returns a merged status string combining PR state and review status.
func combinedStatus(pr PR, rs reviewStatus) (label string, color string) {
	if pr.Draft {
		return "draft", "\033[2m"
	}
	if rs.Total == 0 {
		return "open", "\033[32m"
	}
	if rs.Unresolved == 0 {
		return "approved", "\033[1;32m"
	}
	return fmt.Sprintf("reviewed (%d unresolved)", rs.Unresolved), "\033[33m"
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
		// Skip threads started by ignored reviewers
		if len(t.Comments.Nodes) > 0 && ignore[t.Comments.Nodes[0].Author.Login] {
			continue
		}
		total++
		if !t.IsResolved {
			unresolved++
		}
	}
	return reviewStatus{Unresolved: unresolved, Total: total}
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

func printMySection(title string, prs []PR) {
	fmt.Printf("\n\033[1;36m%s\033[0m (%d件)\n", title, len(prs))

	if len(prs) == 0 {
		fmt.Println("  なし")
		return
	}

	// Sort all PRs by UpdatedAt descending
	for i := 0; i < len(prs); i++ {
		for j := i + 1; j < len(prs); j++ {
			if prs[j].UpdatedAt.After(prs[i].UpdatedAt) {
				prs[i], prs[j] = prs[j], prs[i]
			}
		}
	}

	// Fetch review statuses in parallel
	statusMap := fetchAllReviewStatuses(prs)

	// Table header
	fmt.Printf("\033[1m  %-25s  %-20s  %-8s  %s\033[0m\n",
		"Status", "Repo", "Updated", "Title")
	fmt.Println("  " + strings.Repeat("─", 100))

	for _, pr := range prs {
		repo := truncate(pr.RepoName(), 20)
		prTitle := truncate(pr.Title, 45)
		age := formatAge(pr.UpdatedAt)
		status, statusColor := combinedStatus(pr, statusMap[pr.HTMLURL])
		linkedTitle := fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", pr.HTMLURL, prTitle)
		fmt.Printf("  %s%-25s\033[0m  %-20s  \033[2m%-8s\033[0m  %s\n",
			statusColor, status, repo, age, linkedTitle)
	}
}

func printSection(title string, prs []PR) {
	fmt.Printf("\n\033[1;36m%s\033[0m (%d件)\n", title, len(prs))

	if len(prs) == 0 {
		fmt.Println("  なし")
		return
	}

	// Sort all PRs by UpdatedAt descending
	for i := 0; i < len(prs); i++ {
		for j := i + 1; j < len(prs); j++ {
			if prs[j].UpdatedAt.After(prs[i].UpdatedAt) {
				prs[i], prs[j] = prs[j], prs[i]
			}
		}
	}

	// Table header
	fmt.Printf("\033[1m  %-20s  %-15s  %-8s  %s\033[0m\n",
		"Repo", "Author", "Updated", "Title")
	fmt.Println("  " + strings.Repeat("─", 100))

	for _, pr := range prs {
		repo := truncate(pr.RepoName(), 20)
		title := truncate(pr.Title, 45)
		age := formatAge(pr.UpdatedAt)
		author := truncate(pr.User.Login, 15)
		linkedTitle := fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", pr.HTMLURL, title)
		fmt.Printf("  %-20s  \033[32m%-15s\033[0m  \033[2m%-8s\033[0m  %s\n",
			repo, author, age, linkedTitle)
	}
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

func printHeader(username, org string) {
	header := fmt.Sprintf("\033[1;32m📋 GitHub Dashboard for @%s\033[0m", username)
	if org != "" {
		header += fmt.Sprintf(" \033[2m(org: %s)\033[0m", org)
	}
	fmt.Println(header)
}

type fetchResult struct {
	prs []PR
	err error
}

func cmdMy(username, orgQ string) {
	var authorRes, assigneeRes fetchResult
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		prs, err := ghSearchPRs(fmt.Sprintf("author:%s%s", username, orgQ))
		authorRes = fetchResult{prs, err}
	}()
	go func() {
		defer wg.Done()
		prs, err := ghSearchPRs(fmt.Sprintf("assignee:%s%s", username, orgQ))
		assigneeRes = fetchResult{prs, err}
	}()
	wg.Wait()

	if authorRes.err != nil {
		fmt.Fprintf(os.Stderr, "\033[1;31mError fetching authored PRs: %v\033[0m\n", authorRes.err)
	}
	if assigneeRes.err != nil {
		fmt.Fprintf(os.Stderr, "\033[1;31mError fetching assigned PRs: %v\033[0m\n", assigneeRes.err)
	}
	myPRs := mergePRs(authorRes.prs, assigneeRes.prs)
	printMySection("🔧 自分のPR (作成・アサイン)", myPRs)
}

func cmdBot(org string) {
	repos := splitEnv(os.Getenv("GV_BOT_REPOS"))
	if len(repos) == 0 {
		fmt.Fprintln(os.Stderr, "\033[1;31mGV_BOT_REPOS を設定してください (例: GV_BOT_REPOS=frontend,backend)\033[0m")
		return
	}
	if org == "" {
		fmt.Fprintln(os.Stderr, "\033[1;31mGV_ORG または -org を指定してください\033[0m")
		return
	}

	var allPRs []PR
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, repo := range repos {
		wg.Add(1)
		go func(r string) {
			defer wg.Done()
			fullRepo := fmt.Sprintf("%s/%s", org, r)
			prs, err := ghSearchPRs(fmt.Sprintf("repo:%s", fullRepo))
			if err != nil {
				fmt.Fprintf(os.Stderr, "\033[1;31mError fetching PRs for %s: %v\033[0m\n", fullRepo, err)
				return
			}
			mu.Lock()
			for _, pr := range prs {
				if pr.User.Type == "Bot" {
					allPRs = append(allPRs, pr)
				}
			}
			mu.Unlock()
		}(repo)
	}

	wg.Wait()

	// Deduplicate
	seen := make(map[string]bool)
	var unique []PR
	for _, pr := range allPRs {
		if !seen[pr.HTMLURL] {
			seen[pr.HTMLURL] = true
			unique = append(unique, pr)
		}
	}

	printSection("🤖 Bot PR", unique)
}

// splitEnv splits a comma-separated env value into trimmed non-empty strings.
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

func cmdReview(username, orgQ string) {
	prs, err := ghSearchPRs(fmt.Sprintf("review-requested:%s%s", username, orgQ))
	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[1;31mError fetching review-requested PRs: %v\033[0m\n", err)
		return
	}
	printSection("👀 レビューリクエストされたPR", prs)
}

func usage() {
	fmt.Fprintln(os.Stderr, `Usage: github-viewer [command] [options]

Commands:
  my       自分が作成・アサインされたPRを表示
  review   レビューリクエストされたPRを表示
  bot      Bot PRを表示 (GV_BOT_REPOS + GV_ORG で指定)
  (なし)   両方を表示

Options:
  -org string   Filter by GitHub organization (also via GV_ORG env)`)
	os.Exit(1)
}

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help" || os.Args[1] == "help") {
		usage()
	}

	// Determine subcommand.
	subcmd := ""
	flagArgs := os.Args[1:]
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		subcmd = os.Args[1]
		flagArgs = os.Args[2:]
	}

	fs := flag.NewFlagSet("github-viewer", flag.ExitOnError)
	orgFlag := fs.String("org", "", "Filter by GitHub organization (also via GV_ORG env)")
	fs.Parse(flagArgs)

	org := *orgFlag
	if org == "" {
		org = os.Getenv("GV_ORG")
	}
	orgQ := orgQualifier(org)

	username, err := getUsername()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintln(os.Stderr, "gh CLI がインストールされ、認証済みであることを確認してください。")
		os.Exit(1)
	}

	printHeader(username, org)

	switch subcmd {
	case "my":
		cmdMy(username, orgQ)
	case "review":
		cmdReview(username, orgQ)
	case "bot":
		cmdBot(org)
	case "":
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			cmdMy(username, orgQ)
		}()
		go func() {
			defer wg.Done()
			cmdReview(username, orgQ)
		}()
		wg.Wait()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", subcmd)
		usage()
	}

	fmt.Println()
}
