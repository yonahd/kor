package kor

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var pvcCmd = &cobra.Command{
	Use:   "pvc",
	Short: "Gets unused pvcs",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if outputFormat == "json" || outputFormat == "yaml" {
			if response, err := kor.GetUnusedPvcsStructured(includeExcludeLists, kubeconfig, outputFormat); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(response)
			}
		} else if slackWebhookURL != "" {
			kor.GetUnusedPvcsSendToSlackWebhook(includeExcludeLists, kubeconfig, slackWebhookURL)
		} else if slackChannel != "" && slackAuthToken != "" {
			kor.GetUnusedPvcsSendToSlackAsFile(includeExcludeLists, kubeconfig, slackChannel, slackAuthToken)
		} else {
			kor.GetUnusedPvcs(includeExcludeLists, kubeconfig)
		}
	},
}

func init() {
	rootCmd.AddCommand(pvcCmd)
}
