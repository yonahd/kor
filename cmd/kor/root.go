package kor

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
	"github.com/yonahd/kor/pkg/utils"
)

var rootCmd = &cobra.Command{
	Use:   "kor",
	Short: "kor - a CLI to to discover unused Kubernetes resources",
	Long: `kor is a CLI to to discover unused Kubernetes resources
	kor can currently discover unused configmaps and secrets`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		resourceNames := args[0]
		clientset := kor.GetKubeClient(kubeconfig)
		apiExtClient := kor.GetAPIExtensionsClient(kubeconfig)
		dynamicClient := kor.GetDynamicClient(kubeconfig)

		if response, err := kor.GetUnusedMulti(includeExcludeLists, resourceNames, filterOptions, clientset, apiExtClient, dynamicClient, outputFormat, opts); err != nil {
			fmt.Println(err)
		} else {
			utils.PrintLogo(outputFormat)
			fmt.Println(response)
		}
	},
}

var (
	outputFormat        string
	kubeconfig          string
	includeExcludeLists kor.IncludeExcludeLists
	opts                kor.Opts
	filterOptions       = kor.NewFilterOptions()
)

func Execute() {
	rootCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "Path to kubeconfig file (optional)")
	rootCmd.PersistentFlags().StringVarP(&includeExcludeLists.IncludeListStr, "include-namespaces", "n", "", "Namespaces to run on, splited by comma. Example: --include-namespace ns1,ns2,ns3. ")
	rootCmd.PersistentFlags().StringVarP(&includeExcludeLists.ExcludeListStr, "exclude-namespaces", "e", "", "Namespaces to be excluded, splited by comma. Example: --exclude-namespace ns1,ns2,ns3. If --include-namespace is set, --exclude-namespaces will be ignored.")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, json or yaml)")
	rootCmd.PersistentFlags().StringVar(&opts.WebhookURL, "slack-webhook-url", "", "Slack webhook URL to send notifications to")
	rootCmd.PersistentFlags().StringVar(&opts.Channel, "slack-channel", "", "Slack channel to send notifications to. --slack-channel requires --slack-auth-token to be set.")
	rootCmd.PersistentFlags().StringVar(&opts.Token, "slack-auth-token", "", "Slack auth token to send notifications to. --slack-auth-token requires --slack-channel to be set.")
	rootCmd.PersistentFlags().BoolVar(&opts.DeleteFlag, "delete", false, "Delete unused resources")
	rootCmd.PersistentFlags().BoolVar(&opts.NoInteractive, "no-interactive", false, "Do not prompt for confirmation when deleting resources. Be careful using this flag!")
	rootCmd.PersistentFlags().BoolVarP(&opts.NoNamespaced, "include-non-namespaced", "i", false, "Include recources not bound to a namespace.")
	rootCmd.PersistentFlags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Verbose output (print empty namespaces)")
	addFilterOptionsFlag(rootCmd, filterOptions)

	if err := filterOptions.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Error while validating filter options '%s'", err)
		os.Exit(1)
	}
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error while executing your CLI '%s'", err)
		os.Exit(1)
	}
}

func addFilterOptionsFlag(cmd *cobra.Command, opts *kor.FilterOptions) {
	cmd.PersistentFlags().StringVarP(&opts.ExcludeLabels, "exclude-labels", "l", opts.ExcludeLabels, "Selector to filter out, Example: --exclude-labels key1=value1,key2=value2.")
	cmd.PersistentFlags().StringVar(&opts.NewerThan, "newer-than", opts.NewerThan, "The maximum age of the resources to be considered unused. This flag cannot be used together with older-than flag. Example: --newer-than=1h2m")
	cmd.PersistentFlags().StringVar(&opts.OlderThan, "older-than", opts.OlderThan, "The minimum age of the resources to be considered unused. This flag cannot be used together with newer-than flag. Example: --older-than=1h2m")
}
