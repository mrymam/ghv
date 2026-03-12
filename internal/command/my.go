package command

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/mrymam/ghv/internal/clipboard"
	"github.com/mrymam/ghv/internal/format"
	gh "github.com/mrymam/ghv/pkg/github"
	"github.com/spf13/cobra"
)

// NewMyCmd creates the my subcommand.
func NewMyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "my",
		Short: "自分が作成・アサインされたPRを表示",
		RunE: func(cmd *cobra.Command, args []string) error {
			org, orgQ := ResolveOrg()
			username, err := gh.GetUsername()
			if err != nil {
				return err
			}
			format.PrintHeader(username, org)
			cmdMy(username, orgQ)
			PostRun(cmd)
			return nil
		},
	}
}

func cmdMy(username, orgQ string) {
	var authorRes, assigneeRes gh.FetchResult
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		prs, err := gh.SearchPRs(fmt.Sprintf("author:%s%s", username, orgQ))
		authorRes = gh.FetchResult{PRs: prs, Err: err}
	}()
	go func() {
		defer wg.Done()
		prs, err := gh.SearchPRs(fmt.Sprintf("assignee:%s%s", username, orgQ))
		assigneeRes = gh.FetchResult{PRs: prs, Err: err}
	}()
	wg.Wait()

	if authorRes.Err != nil {
		fmt.Fprintf(os.Stderr, "\033[1;31mError fetching authored PRs: %v\033[0m\n", authorRes.Err)
	}
	if assigneeRes.Err != nil {
		fmt.Fprintf(os.Stderr, "\033[1;31mError fetching assigned PRs: %v\033[0m\n", assigneeRes.Err)
	}
	myPRs := gh.MergePRs(authorRes.PRs, assigneeRes.PRs)
	printMySection("🔧 自分のPR (作成・アサイン)", myPRs)
}

func printMySection(title string, prs []gh.PR) {
	fmt.Printf("\n\033[1;36m%s\033[0m (%d件)\n", title, len(prs))

	if len(prs) == 0 {
		fmt.Println("  なし")
		return
	}

	gh.SortPRsByUpdated(prs)

	// Fetch review statuses in parallel
	ignore := gh.ParseIgnoredReviewers(os.Getenv("GHV_IGNORE_REVIEWERS"))
	statusMap := gh.FetchAllReviewStatuses(prs, ignore)

	// Table header
	fmt.Printf("\033[1m  %-25s  %-20s  %-8s  %s\033[0m\n",
		"Status", "Repo", "Updated", "Title")
	fmt.Println("  " + strings.Repeat("─", 100))

	for _, pr := range prs {
		repo := format.Truncate(pr.RepoName(), 20)
		prTitle := format.Truncate(pr.Title, 45)
		age := format.FormatAge(pr.UpdatedAt)
		status, statusColor := gh.CombinedStatus(pr, statusMap[pr.HTMLURL])
		linkedTitle := fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", pr.HTMLURL, prTitle)
		fmt.Printf("  %s%-25s\033[0m  %-20s  \033[2m%-8s\033[0m  %s\n",
			statusColor, status, repo, age, linkedTitle)
	}
	items := make([]clipboard.ClipboardItem, len(prs))
	for i, pr := range prs {
		items[i] = clipboard.ClipboardItem{Title: pr.Title, URL: pr.HTMLURL}
	}
	clipboard.AppendClipboard(items)
}
