package kor

import (
	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var pvcCmd = &cobra.Command{
	Use:   "pvc",
	Short: "Gets unused pvcs",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if outputFormat == "json" {
			kor.GetUnusedPvcsJson(namespace, kubeconfig)
		} else {
			kor.GetUnusedPvcs(namespace, kubeconfig)
		}
	},
}

func init() {
	rootCmd.AddCommand(pvcCmd)
}
