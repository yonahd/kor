package kor

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var rootCmd = &cobra.Command{
	Use:   "kor",
	Short: "kor - a CLI to to discover unused Kubernetes resources",
	Long: `kor is a CLI to to discover unused Kubernetes resources
	kor can currently discover unused configmaps and secrets`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		resourceNames := args[0]

		// Cheks whether the string contains a comma, indicating that it represents a list of resources
		if strings.ContainsRune(resourceNames, 44) {
			if outputFormat == "json" || outputFormat == "yaml" {
				if response, err := kor.GetUnusedMultiStructured(includeExcludeLists, kubeconfig, outputFormat, resourceNames); err != nil {
					fmt.Println(err)
				} else {
					fmt.Println(response)
				}
			} else {
				kor.GetUnusedMulti(includeExcludeLists, kubeconfig, resourceNames)
			}
		} else {
			fmt.Printf("Subcommand %q was not found, try using 'kor --help' for available subcommands", args[0])
		}
	},
}

var (
	outputFormat        string
	kubeconfig          string
	slackWebhookURL     string
	slackChannel        string
	slackAuthToken      string
	includeExcludeLists kor.IncludeExcludeLists
)

func Execute() {
	rootCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "Path to kubeconfig file (optional)")
	rootCmd.PersistentFlags().StringVarP(&includeExcludeLists.IncludeListStr, "include-namespaces", "n", "", "Namespaces to run on, splited by comma. Example: --include-namespace ns1,ns2,ns3. ")
	rootCmd.PersistentFlags().StringVarP(&includeExcludeLists.ExcludeListStr, "exclude-namespaces", "e", "", "Namespaces to be excluded, splited by comma. Example: --exclude-namespace ns1,ns2,ns3. If --include-namespace is set, --exclude-namespaces will be ignored.")
	rootCmd.PersistentFlags().StringVar(&outputFormat, "output", "table", "Output format (table or json)")
	rootCmd.PersistentFlags().StringVar(&slackWebhookURL, "slack-webhook-url", "", "Slack webhook URL to send notifications to")
	rootCmd.PersistentFlags().StringVar(&slackChannel, "slack-channel", "", "Slack channel to send notifications to")
	rootCmd.PersistentFlags().StringVar(&slackAuthToken, "slack-auth-token", "", "Slack auth token to send notifications to")
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error while executing your CLI '%s'", err)
		os.Exit(1)
	}
}
