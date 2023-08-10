package kor

import (
	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var allCmd = &cobra.Command{
	Use:   "all",
	Short: "Gets unused resources",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if outputFormat == "json" {
			kor.GetUnusedAllJSON(namespace, kubeconfig)
		} else {
			kor.GetUnusedAll(namespace, kubeconfig)
		}

	},
}

func init() {
	rootCmd.AddCommand(allCmd)
}
