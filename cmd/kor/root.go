package kor

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kor",
	Short: "kor - a CLI to to discover unused Kubernetes resources",
	Long: `kor is a CLI to to discover unused Kubernetes resources
	kor can currently discover unused configmaps and secrets`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

	},
}

var namespace string
var outputFormat string
var kubeconfig string
var slackWebhookURL string

func Execute() {
	rootCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "Path to kubeconfig file (optional)")
	rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "", "Namespace to run on")
	rootCmd.PersistentFlags().StringVar(&outputFormat, "output", "table", "Output format (table or json)")
	rootCmd.PersistentFlags().StringVar(&slackWebhookURL, "slack-webhook-url", "", "Slack webhook URL to send notifications to")
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error while executing your CLI '%s'", err)
		os.Exit(1)
	}
}
