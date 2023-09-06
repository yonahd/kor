package kor

import (
	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var ingressCmd = &cobra.Command{
	Use:     "ingress",
	Aliases: []string{"ing"},
	Short:   "Gets unused ingresses",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if outputFormat == "json" {
			kor.GetUnusedIngressesJSON(namespace, kubeconfig)
		} else if slackWebhookURL != "" {
			kor.GetUnusedIngressesSlack(namespace, kubeconfig, slackWebhookURL)
		} else {
			kor.GetUnusedIngresses(namespace, kubeconfig)
		}

	},
}

func init() {
	rootCmd.AddCommand(ingressCmd)
}
