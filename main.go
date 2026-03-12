package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
)

func usage() {
	fmt.Fprintln(os.Stderr, `Usage: gv [command] [options]

Commands:
  my       自分が作成・アサインされたPRを表示
  review   レビューリクエストされたPRを表示
  bot      Bot PRを表示 (GV_BOT_REPOS + GV_ORG で指定)
  (なし)   両方を表示

Options:
  -org string    Filter by GitHub organization (also via GV_ORG env)
  -copy          Copy PR list as markdown links to clipboard`)
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

	fs := flag.NewFlagSet("gv", flag.ExitOnError)
	orgFlag := fs.String("org", "", "Filter by GitHub organization (also via GV_ORG env)")
	copyFlag := fs.Bool("copy", false, "Copy PR list as markdown links to clipboard")
	fs.Parse(flagArgs)

	org := *orgFlag
	if org == "" {
		org = os.Getenv("GV_ORG")
	}
	orgQ := orgQualifier(org)
	doCopy := *copyFlag
	if !flagSet(fs, "copy") {
		if v := os.Getenv("GV_DEFAULT_COPY_ON"); v != "" {
			doCopy = strings.EqualFold(v, "true")
		}
	}

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

	if doCopy {
		flushClipboard()
	}
	fmt.Println()
}
