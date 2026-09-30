package cmd

import (
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate commit message tanpa mengeksekusi commit",
	Args:  cobra.NoArgs,
	RunE:  generateOnly,
}
