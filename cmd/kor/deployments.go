package kor

import (
	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var deployCmd = &cobra.Command{
	Use:     "deployments",
	Aliases: []string{"deploy"},
	Short:   "Gets unused deployments",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if outputFormat == "json" {
			kor.GetUnusedDeploymentsJSON(includeExcludeLists, kubeconfig)
		} else {
			kor.GetUnusedDeployments(includeExcludeLists, kubeconfig)
		}

	},
}

func init() {
	rootCmd.AddCommand(deployCmd)
}
