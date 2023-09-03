package kor

import (
	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var roleCmd = &cobra.Command{
	Use:   "role",
	Short: "Gets unused roles",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if outputFormat == "json" {
			kor.GetUnusedRolesJSON(includeExcludeLists, kubeconfig)
		} else {
			kor.GetUnusedRoles(includeExcludeLists, kubeconfig)
		}
	},
}

func init() {
	rootCmd.AddCommand(roleCmd)
}
