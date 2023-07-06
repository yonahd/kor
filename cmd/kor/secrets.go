package kor

import (
	"github.com/spf13/cobra"
	"github.com/ydissen/kor/pkg/kor"
)

var secretCmd = &cobra.Command{
	Use:     "secret",
	Aliases: []string{"scrt"},
	Short:   "Gets unused secrets",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		kor.GetUnusedSecrets()

	},
}

func init() {
	rootCmd.AddCommand(secretCmd)
}
