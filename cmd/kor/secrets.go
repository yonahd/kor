package kor

import (
	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var secretCmd = &cobra.Command{
	Use:     "secret",
	Aliases: []string{"scrt"},
	Short:   "Gets unused secrets",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if outputFormat == "json" {
			kor.GetUnusedSecretsJSON(namespace, kubeconfig)
		} else if slackWebhookURL != "" {
			kor.GetUnusedSecretsSendToSlackWebhook(namespace, kubeconfig, slackWebhookURL)
		} else if slackChannel != "" && slackAuthToken != "" {
			kor.GetUnusedSecretsSendToSlackAsFile(namespace, kubeconfig, slackChannel, slackAuthToken)
		} else {
			kor.GetUnusedSecrets(namespace, kubeconfig)
		}

	},
}

func init() {
	rootCmd.AddCommand(secretCmd)
}
