package kor

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var stsCmd = &cobra.Command{
	Use:     "statefulset",
	Aliases: []string{"sts", "statefulsets"},
	Short:   "Gets unused statefulSets",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		clientset := kor.GetKubeClient(kubeconfig)
		if outputFormat == "json" || outputFormat == "yaml" {
			if response, err := kor.GetUnusedStatefulSetsStructured(includeExcludeLists, clientset, outputFormat); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(response)
			}
		} else if slackWebhookURL != "" {
			kor.GetUnusedStatefulSets(includeExcludeLists, clientset, slackWebhookURL)
		} else if slackChannel != "" && slackAuthToken != "" {
			kor.GetUnusedStatefulSets(includeExcludeLists, clientset, slackChannel, slackAuthToken)
		} else {
			kor.GetUnusedStatefulSets(includeExcludeLists, clientset)
		}

	},
}

func init() {
	rootCmd.AddCommand(stsCmd)
}
