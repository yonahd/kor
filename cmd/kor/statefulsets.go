package kor

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var stsCmd = &cobra.Command{
	Use:     "statefulset",
	Aliases: []string{"sts", "statefulsets"},
	Short:   "Gets unused statefulsets",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if outputFormat == "json" || outputFormat == "yaml" {
			if response, err := kor.GetUnusedStatefulsetsStructured(includeExcludeLists, kubeconfig, outputFormat); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(response)
			}
		} else if slackWebhookURL != "" {
			kor.GetUnusedStatefulsetsSendToSlackWebhook(includeExcludeLists, kubeconfig, slackWebhookURL)
		} else if slackChannel != "" && slackAuthToken != "" {
			kor.GetUnusedStatefulsetsSendToSlackAsFile(includeExcludeLists, kubeconfig, slackChannel, slackAuthToken)
		} else {
			kor.GetUnusedStatefulsets(includeExcludeLists, kubeconfig)
		}

	},
}

func init() {
	rootCmd.AddCommand(stsCmd)
}
