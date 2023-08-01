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
			kor.GetUnusedServiceAccountsJSON(namespace, kubeconfig)
		} else {
			kor.GetUnusedServiceAccounts(namespace, kubeconfig)
		}

	},
}

func init() {
	serviceAccountCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "Path to kubeconfig file (optional)")
	serviceAccountCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "", "Namespace to run on")
	serviceAccountCmd.PersistentFlags().StringVar(&outputFormat, "output", "table", "Output format (table or json)")
	rootCmd.AddCommand(serviceAccountCmd)
}
