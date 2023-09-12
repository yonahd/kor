package kor

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var configmapCmd = &cobra.Command{
	Use:     "configmap",
	Aliases: []string{"cm", "configmaps"},
	Short:   "Gets unused configmaps",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if outputFormat == "json" || outputFormat == "yaml" {
			if response, err := kor.GetUnusedConfigmapsStructured(includeExcludeLists, kubeconfig, outputFormat); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(response)
			}
		} else if slackWebhookURL != "" {
			kor.GetUnusedConfigmapsSendToSlackWebhook(includeExcludeLists, kubeconfig, slackWebhookURL)
		} else if slackChannel != "" && slackAuthToken != "" {
			kor.GetUnusedConfigmapsSendToSlackAsFile(includeExcludeLists, kubeconfig, slackChannel, slackAuthToken)
		} else {
			kor.GetUnusedConfigmaps(includeExcludeLists, kubeconfig)
		}

	},
}

func init() {
	rootCmd.AddCommand(configmapCmd)
}
