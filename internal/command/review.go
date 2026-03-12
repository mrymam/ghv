package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/mrymam/ghv/internal/clipboard"
	"github.com/mrymam/ghv/internal/format"
	gh "github.com/mrymam/ghv/pkg/github"
	"github.com/spf13/cobra"
)

// NewReviewCmd creates the review subcommand.
func NewReviewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "review",
		Short: "レビューリクエストされたPRを表示",
		RunE: func(cmd *cobra.Command, args []string) error {
			org, orgQ := ResolveOrg()
			username, err := gh.GetUsername()
			if err != nil {
				return err
			}
			format.PrintHeader(username, org)
			cmdReview(username, orgQ)
			PostRun(cmd)
			return nil
		},
	}
}

func cmdReview(username, orgQ string) {
	prs, err := gh.SearchPRs(fmt.Sprintf("review-requested:%s%s", username, orgQ))
	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[1;31mError fetching review-requested PRs: %v\033[0m\n", err)
		return
	}
	printSection("👀 レビューリクエストされたPR", prs)
}

func printSection(title string, prs []gh.PR) {
	fmt.Printf("\n\033[1;36m%s\033[0m (%d件)\n", title, len(prs))

	if len(prs) == 0 {
		fmt.Println("  なし")
		return
	}

	gh.SortPRsByUpdated(prs)

	// Table header
	fmt.Printf("\033[1m  %-20s  %-15s  %-8s  %s\033[0m\n",
		"Repo", "Author", "Updated", "Title")
	fmt.Println("  " + strings.Repeat("─", 100))

	for _, pr := range prs {
		repo := format.Truncate(pr.RepoName(), 20)
		title := format.Truncate(pr.Title, 45)
		age := format.FormatAge(pr.UpdatedAt)
		author := format.Truncate(pr.User.Login, 15)
		linkedTitle := fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", pr.HTMLURL, title)
		fmt.Printf("  %-20s  \033[32m%-15s\033[0m  \033[2m%-8s\033[0m  %s\n",
			repo, author, age, linkedTitle)
	}
	items := make([]clipboard.ClipboardItem, len(prs))
	for i, pr := range prs {
		items[i] = clipboard.ClipboardItem{Title: pr.Title, URL: pr.HTMLURL}
	}
	clipboard.AppendClipboard(items)
}
