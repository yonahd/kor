package kor

import (
	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var configmapCmd = &cobra.Command{
	Use:     "configmap",
	Aliases: []string{"cm"},
	Short:   "Gets unused configmaps",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if outputFormat == "json" {
			kor.GetUnusedConfigmapsJSON(namespace, kubeconfig)
		} else if slackWebhookURL != "" {
			kor.GetUnusedConfigmapsSendToSlackWebhook(namespace, kubeconfig, slackWebhookURL)
		} else if slackChannel != "" && slackAuthToken != "" {
			kor.GetUnusedConfigmapsSendToSlackAsFile(namespace, kubeconfig, slackChannel, slackAuthToken)
		} else {
			kor.GetUnusedConfigmaps(namespace, kubeconfig)
		}

	},
}

func init() {
	rootCmd.AddCommand(configmapCmd)
}
