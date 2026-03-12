package main

import (
	"fmt"
	"os"
	"sync"
)

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
