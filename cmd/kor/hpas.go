package kor

import (
	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var hpaCmd = &cobra.Command{
	Use:   "hpa",
	Short: "Gets unused hpas",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if outputFormat == "json" {
			kor.GetUnusedHpasJson(namespace, kubeconfig)
		} else if slackWebhookURL != "" {
			kor.GetUnusedHpasSendToSlackWebhook(namespace, kubeconfig, slackWebhookURL)
		} else if slackChannel != "" && slackAuthToken != "" {
			kor.GetUnusedHpasSendToSlackAsFile(namespace, kubeconfig, slackChannel, slackAuthToken)
		} else {
			kor.GetUnusedHpas(namespace, kubeconfig)
		}
	},
}

func init() {
	rootCmd.AddCommand(hpaCmd)
}
