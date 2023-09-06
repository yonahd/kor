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
			kor.GetUnusedDeploymentsJSON(namespace, kubeconfig)
		} else if slackWebhookURL != "" {
			kor.GetUnusedDeploymentsSlack(namespace, kubeconfig, slackWebhookURL)
		} else {
			kor.GetUnusedDeployments(namespace, kubeconfig)
		}

	},
}

func init() {
	rootCmd.AddCommand(deployCmd)
}
