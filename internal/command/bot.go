package command

import (
	"fmt"
	"os"
	"sync"

	"github.com/mrymam/ghv/internal/format"
	gh "github.com/mrymam/ghv/pkg/github"
	"github.com/spf13/cobra"
)

// NewBotCmd creates the bot subcommand.
func NewBotCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bot",
		Short: "Bot PRを表示 (GHV_BOT_REPOS + GHV_ORG で指定)",
		RunE: func(cmd *cobra.Command, args []string) error {
			org, _ := ResolveOrg()
			username, err := gh.GetUsername()
			if err != nil {
				return err
			}
			format.PrintHeader(username, org)
			cmdBot(org)
			PostRun(cmd)
			return nil
		},
	}
}

func cmdBot(org string) {
	repos := format.SplitEnv(os.Getenv("GHV_BOT_REPOS"))
	if len(repos) == 0 {
		fmt.Fprintln(os.Stderr, "\033[1;31mGHV_BOT_REPOS を設定してください (例: GHV_BOT_REPOS=frontend,backend)\033[0m")
		return
	}
	if org == "" {
		fmt.Fprintln(os.Stderr, "\033[1;31mGHV_ORG または -org を指定してください\033[0m")
		return
	}

	var allPRs []gh.PR
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, repo := range repos {
		wg.Add(1)
		go func(r string) {
			defer wg.Done()
			fullRepo := fmt.Sprintf("%s/%s", org, r)
			prs, err := gh.SearchPRs(fmt.Sprintf("repo:%s", fullRepo))
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
	var unique []gh.PR
	for _, pr := range allPRs {
		if !seen[pr.HTMLURL] {
			seen[pr.HTMLURL] = true
			unique = append(unique, pr)
		}
	}

	printSection("🤖 Bot PR", unique)
}
