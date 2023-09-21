package kor

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var hpaCmd = &cobra.Command{
	Use:     "horizontalpodautoscaler",
	Aliases: []string{"hpa", "horizontalpodautoscalers"},
	Short:   "Gets unused hpas",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		clientset := kor.GetKubeClient(kubeconfig)
		if outputFormat == "json" || outputFormat == "yaml" {
			if response, err := kor.GetUnusedHpasStructured(includeExcludeLists, clientset, outputFormat); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(response)
			}
		} else if slackWebhookURL != "" {
			kor.GetUnusedHpas(includeExcludeLists, clientset, slackWebhookURL)
		} else if slackChannel != "" && slackAuthToken != "" {
			kor.GetUnusedHpas(includeExcludeLists, clientset, slackChannel, slackAuthToken)
		} else {
			kor.GetUnusedHpas(includeExcludeLists, clientset)
		}
	},
}

func init() {
	rootCmd.AddCommand(hpaCmd)
}
