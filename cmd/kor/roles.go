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
			kor.GetUnusedRolesJSON(namespace, kubeconfig)
		} else if slackWebhookURL != "" {
			kor.GetUnusedRolesSendToSlackWebhook(namespace, kubeconfig, slackWebhookURL)
		} else if slackChannel != "" && slackAuthToken != "" {
			kor.GetUnusedRolesSendToSlackAsFile(namespace, kubeconfig, slackChannel, slackAuthToken)
		} else {
			kor.GetUnusedRoles(namespace, kubeconfig)
		}
	},
}

func init() {
	rootCmd.AddCommand(roleCmd)
}
