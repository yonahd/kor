package kor

import (
	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var serviceAccountCmd = &cobra.Command{
	Use:     "serviceaccount",
	Aliases: []string{"sa"},
	Short:   "Gets unused service accounts",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if outputFormat == "json" {
			kor.GetUnusedServiceAccountsJSON(includeExcludeLists, kubeconfig)
		} else {
			kor.GetUnusedServiceAccounts(includeExcludeLists, kubeconfig)
		}

	},
}

func init() {
	rootCmd.AddCommand(serviceAccountCmd)
}
