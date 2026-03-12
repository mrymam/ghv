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

var (
	orgFlag  string
	copyFlag bool
)

// RegisterFlags adds persistent flags to the root command.
func RegisterFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&orgFlag, "org", "", "Filter by GitHub organization (also via GHV_ORG env)")
	cmd.PersistentFlags().BoolVar(&copyFlag, "copy", false, "Copy PR list as rich text links to clipboard")
}

// ResolveOrg returns the org name and its search qualifier.
func ResolveOrg() (string, string) {
	org := orgFlag
	if org == "" {
		org = os.Getenv("GHV_ORG")
	}
	return org, format.OrgQualifier(org)
}

// ResolveCopy determines whether clipboard copy is enabled.
func ResolveCopy(cmd *cobra.Command) bool {
	if cmd.Flags().Changed("copy") {
		return copyFlag
	}
	if v := os.Getenv("GHV_DEFAULT_COPY_ON"); v != "" {
		return strings.EqualFold(v, "true")
	}
	return false
}

// PostRun performs post-command actions (clipboard flush, trailing newline).
func PostRun(cmd *cobra.Command) {
	if ResolveCopy(cmd) {
		clipboard.FlushClipboard()
	}
	fmt.Println()
}

// RunDefault runs the default command (My + Review in parallel).
func RunDefault(cmd *cobra.Command, args []string) error {
	org, orgQ := ResolveOrg()
	username, err := gh.GetUsername()
	if err != nil {
		return err
	}
	format.PrintHeader(username, org)

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

	PostRun(cmd)
	return nil
}
