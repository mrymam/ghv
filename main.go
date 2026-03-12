package main

import (
	"os"

	"github.com/mrymam/ghv/internal/command"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:     "ghv",
		Short:   "GitHub PR dashboard CLI",
		Long:    "GitHub上の自分に関連するPRをターミナルで表形式に一覧表示するCLIツール。",
		Version: command.GetVersion(),
		RunE:    command.RunDefault,
	}
	command.RegisterFlags(root)
	root.AddCommand(
		command.NewMyCmd(),
		command.NewReviewCmd(),
		command.NewBotCmd(),
		command.NewTUICmd(),
		command.NewNotifyCmd(),
		command.NewVersionCmd(),
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
