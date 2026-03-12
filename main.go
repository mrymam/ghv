package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/spf13/cobra"
)

var (
	version  = ""
	orgFlag  string
	copyFlag bool
)

func getVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "ghv",
		Short:   "GitHub PR dashboard CLI",
		Long:    "GitHub上の自分に関連するPRをターミナルで表形式に一覧表示するCLIツール。",
		Version: getVersion(),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDefault(cmd)
		},
	}

	root.PersistentFlags().StringVar(&orgFlag, "org", "", "Filter by GitHub organization (also via GHV_ORG env)")
	root.PersistentFlags().BoolVar(&copyFlag, "copy", false, "Copy PR list as rich text links to clipboard")

	root.AddCommand(newMyCmd())
	root.AddCommand(newReviewCmd())
	root.AddCommand(newBotCmd())
	root.AddCommand(newTUICmd())
	root.AddCommand(newNotifyCmd())
	root.AddCommand(newVersionCmd())

	return root
}

func resolveOrg() (string, string) {
	org := orgFlag
	if org == "" {
		org = os.Getenv("GHV_ORG")
	}
	return org, orgQualifier(org)
}

func resolveCopy(cmd *cobra.Command) bool {
	if cmd.Flags().Changed("copy") {
		return copyFlag
	}
	if v := os.Getenv("GHV_DEFAULT_COPY_ON"); v != "" {
		return strings.EqualFold(v, "true")
	}
	return false
}

func postRun(cmd *cobra.Command) {
	if resolveCopy(cmd) {
		flushClipboard()
	}
	fmt.Println()
}

func newMyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "my",
		Short: "自分が作成・アサインされたPRを表示",
		RunE: func(cmd *cobra.Command, args []string) error {
			org, orgQ := resolveOrg()
			username, err := getUsername()
			if err != nil {
				return err
			}
			printHeader(username, org)
			cmdMy(username, orgQ)
			postRun(cmd)
			return nil
		},
	}
}

func newReviewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "review",
		Short: "レビューリクエストされたPRを表示",
		RunE: func(cmd *cobra.Command, args []string) error {
			org, orgQ := resolveOrg()
			username, err := getUsername()
			if err != nil {
				return err
			}
			printHeader(username, org)
			cmdReview(username, orgQ)
			postRun(cmd)
			return nil
		},
	}
}

func newBotCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bot",
		Short: "Bot PRを表示 (GHV_BOT_REPOS + GHV_ORG で指定)",
		RunE: func(cmd *cobra.Command, args []string) error {
			org, _ := resolveOrg()
			username, err := getUsername()
			if err != nil {
				return err
			}
			printHeader(username, org)
			cmdBot(org)
			postRun(cmd)
			return nil
		},
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "バージョンを表示",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("ghv version %s\n", getVersion())
		},
	}
}

func runDefault(cmd *cobra.Command) error {
	org, orgQ := resolveOrg()
	username, err := getUsername()
	if err != nil {
		return err
	}
	printHeader(username, org)

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

	postRun(cmd)
	return nil
}

func main() {
	root := newRootCmd()
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
