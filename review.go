package main

import (
	"fmt"
	"os"
	"strings"
)

func cmdReview(username, orgQ string) {
	prs, err := ghSearchPRs(fmt.Sprintf("review-requested:%s%s", username, orgQ))
	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[1;31mError fetching review-requested PRs: %v\033[0m\n", err)
		return
	}
	printSection("👀 レビューリクエストされたPR", prs)
}

func printSection(title string, prs []PR) {
	fmt.Printf("\n\033[1;36m%s\033[0m (%d件)\n", title, len(prs))

	if len(prs) == 0 {
		fmt.Println("  なし")
		return
	}

	sortPRsByUpdated(prs)

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
	items := make([]clipboardItem, len(prs))
	for i, pr := range prs {
		items[i] = clipboardItem{Title: pr.Title, URL: pr.HTMLURL}
	}
	appendClipboard(items)
}
