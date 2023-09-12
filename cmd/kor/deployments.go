package kor

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var deployCmd = &cobra.Command{
	Use:     "deployment",
	Aliases: []string{"deploy", "deployments"},
	Short:   "Gets unused deployments",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if outputFormat == "json" || outputFormat == "yaml" {
			if response, err := kor.GetUnusedDeploymentsStructured(includeExcludeLists, kubeconfig, outputFormat); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(response)
			}
		} else if slackWebhookURL != "" {
			kor.GetUnusedDeploymentsSendToSlackWebhook(includeExcludeLists, kubeconfig, slackWebhookURL)
		} else if slackChannel != "" && slackAuthToken != "" {
			kor.GetUnusedDeploymentsSendToSlackAsFile(includeExcludeLists, kubeconfig, slackChannel, slackAuthToken)
		} else {
			kor.GetUnusedDeployments(includeExcludeLists, kubeconfig)
		}

	},
}

func init() {
	rootCmd.AddCommand(deployCmd)
}
