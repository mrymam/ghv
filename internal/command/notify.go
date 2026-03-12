package command

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	gh "github.com/mrymam/ghv/pkg/github"
	"github.com/spf13/cobra"
)

// NewNotifyCmd creates the notify subcommand.
func NewNotifyCmd() *cobra.Command {
	var polling time.Duration

	cmd := &cobra.Command{
		Use:   "notify",
		Short: "新しいレビューリクエストをmacOS通知で知らせる",
		Long:  "定期的にレビューリクエストされたPRをポーリングし、新しいPRが見つかったらmacOS通知を送信する。",
		RunE: func(cmd *cobra.Command, args []string) error {
			if polling < 10*time.Second {
				return fmt.Errorf("polling interval must be at least 10s, got: %s", polling)
			}
			return runWatch(polling)
		},
	}

	cmd.Flags().DurationVar(&polling, "polling", 5*time.Minute, "Polling interval (e.g. 3m, 10m, 1h)")

	return cmd
}

func runWatch(interval time.Duration) error {
	_, orgQ := ResolveOrg()
	username, err := gh.GetUsername()
	if err != nil {
		return err
	}

	fmt.Printf("👀 Watching review requests for @%s (polling every %s)\n", username, interval)
	fmt.Println("Press Ctrl+C to stop.")

	known := make(map[string]bool)

	// Initial fetch to populate known PRs
	prs, err := gh.SearchPRs(fmt.Sprintf("review-requested:%s%s", username, orgQ))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: initial fetch failed: %v\n", err)
	} else {
		for _, pr := range prs {
			known[pr.HTMLURL] = true
		}
		fmt.Printf("Tracking %d existing review request(s).\n", len(known))
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		prs, err := gh.SearchPRs(fmt.Sprintf("review-requested:%s%s", username, orgQ))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Poll error: %v\n", err)
			continue
		}

		current := make(map[string]bool)
		for _, pr := range prs {
			current[pr.HTMLURL] = true
			if !known[pr.HTMLURL] {
				sendNotification(pr)
			}
		}
		known = current

		now := time.Now().Format("15:04:05")
		fmt.Printf("[%s] %d review request(s)\n", now, len(prs))
	}

	return nil
}

func sendNotification(pr gh.PR) {
	title := "New Review Request"
	subtitle := fmt.Sprintf("%s by %s", pr.RepoName(), pr.User.Login)
	message := pr.Title

	args := []string{
		"-title", title,
		"-subtitle", subtitle,
		"-message", message,
		"-open", pr.HTMLURL,
		"-sound", "Glass",
		"-group", "ghv-watch",
	}

	if err := exec.Command("terminal-notifier", args...).Run(); err != nil {
		// Fallback to osascript if terminal-notifier is not available
		script := fmt.Sprintf(
			`display notification %q with title %q sound name "Glass"`,
			subtitle+": "+message, title,
		)
		exec.Command("osascript", "-e", script).Run()
	}

	fmt.Printf("🔔 %s: %s (%s)\n", pr.RepoName(), pr.Title, pr.HTMLURL)
}
