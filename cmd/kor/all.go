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
			kor.GetUnusedAllJSON(namespace)
		} else {
			kor.GetUnusedAll(namespace)
		}

	},
}

func init() {
	allCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "", "Namespace to run on")
	allCmd.PersistentFlags().StringVar(&outputFormat, "output", "table", "Output format (table or json)")
	rootCmd.AddCommand(allCmd)
}
