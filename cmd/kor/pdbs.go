package kor

import (
	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var pdbCmd = &cobra.Command{
	Use:   "pdb",
	Short: "Gets unused pdbs",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if outputFormat == "json" {
			kor.GetUnusedPdbsJson(namespace, kubeconfig)
		} else {
			kor.GetUnusedPdbs(namespace, kubeconfig)
		}
	},
}

func init() {
	rootCmd.AddCommand(pdbCmd)
}
