package kor

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var pdbCmd = &cobra.Command{
	Use:     "poddisruptionbudget",
	Aliases: []string{"pdb", "poddisruptionbudgets"},
	Short:   "Gets unused pdbs",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		clientset := kor.GetKubeClient(kubeconfig)
		if outputFormat == "json" || outputFormat == "yaml" {
			if response, err := kor.GetUnusedPdbsStructured(includeExcludeLists, clientset, outputFormat); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(response)
			}
		} else if slackWebhookURL != "" {
			kor.GetUnusedPdbsSendToSlackWebhook(includeExcludeLists, clientset, slackWebhookURL)
		} else if slackChannel != "" && slackAuthToken != "" {
			kor.GetUnusedPdbsSendToSlackAsFile(includeExcludeLists, clientset, slackChannel, slackAuthToken)
		} else {
			kor.GetUnusedPdbs(includeExcludeLists, clientset)
		}

	},
}

func init() {
	rootCmd.AddCommand(pdbCmd)
}
