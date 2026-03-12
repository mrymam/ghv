package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

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

func printMySection(title string, prs []PR) {
	fmt.Printf("\n\033[1;36m%s\033[0m (%d件)\n", title, len(prs))

	if len(prs) == 0 {
		fmt.Println("  なし")
		return
	}

	sortPRsByUpdated(prs)

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
	items := make([]clipboardItem, len(prs))
	for i, pr := range prs {
		items[i] = clipboardItem{Title: pr.Title, URL: pr.HTMLURL}
	}
	appendClipboard(items)
}
