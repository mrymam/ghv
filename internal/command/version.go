package command

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var version = ""

// GetVersion returns the application version.
func GetVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

// NewVersionCmd creates the version subcommand.
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "バージョンを表示",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("ghv version %s\n", GetVersion())
		},
	}
}
