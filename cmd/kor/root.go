package kor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yonahd/kor/pkg/clusterconfig"
	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
	"github.com/yonahd/kor/pkg/kor"
	"github.com/yonahd/kor/pkg/utils"
)

func execName() string {
	n := "kor"
	if strings.HasPrefix(filepath.Base(os.Args[0]), "kubectl-") {
		return "kubectl-" + n
	}

	return n
}

var rootCmd = &cobra.Command{
	Use:   execName(),
	Short: "kor - a CLI to to discover unused Kubernetes resources",
	Long: `kor is a CLI to to discover unused Kubernetes resources
	kor can currently discover unused configmaps and secrets`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		resourceNames := args[0]
		clientset := clusterconfig.GetKubeClient(kubeconfig)
		clientsetinterface, _ := clusterconfig.GetKubeClientForCrds(kubeconfig, clientset)
		apiExtClient := clusterconfig.GetAPIExtensionsClient(kubeconfig)
		dynamicClient := clusterconfig.GetDynamicClient(kubeconfig)

		if response, err := kor.GetUnusedMulti(resourceNames, filterOptions, clientset, apiExtClient, dynamicClient, clientsetinterface, outputFormat, opts); err != nil {
			fmt.Println(err)
		} else {
			utils.PrintLogo(outputFormat)
			fmt.Println(response)
		}
	},
}

var (
	outputFormat  string
	kubeconfig    string
	opts          common.Opts
	filterOptions = &filters.Options{}
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "Path to kubeconfig file (optional)")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, json or yaml)")
	rootCmd.PersistentFlags().StringVar(&opts.WebhookURL, "slack-webhook-url", "", "Slack webhook URL to send notifications to")
	rootCmd.PersistentFlags().StringVar(&opts.Channel, "slack-channel", "", "Slack channel to send notifications to. --slack-channel requires --slack-auth-token to be set.")
	rootCmd.PersistentFlags().StringVar(&opts.Token, "slack-auth-token", "", "Slack auth token to send notifications to. --slack-auth-token requires --slack-channel to be set.")
	rootCmd.PersistentFlags().BoolVar(&opts.DeleteFlag, "delete", false, "Delete unused resources")
	rootCmd.PersistentFlags().BoolVar(&opts.NoInteractive, "no-interactive", false, "Do not prompt for confirmation when deleting resources. Be careful using this flag!")
	rootCmd.PersistentFlags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Verbose output (print empty namespaces)")
	rootCmd.PersistentFlags().StringVar(&opts.GroupBy, "group-by", "namespace", "Group output by (namespace, resource)")
	rootCmd.PersistentFlags().BoolVar(&opts.ShowReason, "show-reason", false, "Print reason resource is considered unused")
	addFilterOptionsFlag(rootCmd, filterOptions)
}

func Execute() {
	_ = rootCmd.ParseFlags(os.Args)
	if err := filterOptions.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Error while validating filter options '%s'", err)
		os.Exit(1)
	}
	filterOptions.Modify()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error while executing your CLI '%s'", err)
		os.Exit(1)
	}
}

func addFilterOptionsFlag(cmd *cobra.Command, opts *filters.Options) {
	cmd.PersistentFlags().StringSliceVarP(&opts.ExcludeLabels, "exclude-labels", "l", opts.ExcludeLabels, "Selector to filter out, Example: --exclude-labels key1=value1,key2=value2. If --include-labels is set, --exclude-labels will be ignored.")
	cmd.PersistentFlags().StringVar(&opts.NewerThan, "newer-than", opts.NewerThan, "The maximum age of the resources to be considered unused. This flag cannot be used together with older-than flag. Example: --newer-than=1h2m")
	cmd.PersistentFlags().StringVar(&opts.OlderThan, "older-than", opts.OlderThan, "The minimum age of the resources to be considered unused. This flag cannot be used together with newer-than flag. Example: --older-than=1h2m")
	cmd.PersistentFlags().StringVar(&opts.IncludeLabels, "include-labels", opts.IncludeLabels, "Selector to filter in, Example: --include-labels key1=value1.(currently supports one label)")
	cmd.PersistentFlags().StringSliceVarP(&opts.ExcludeNamespaces, "exclude-namespaces", "e", opts.ExcludeNamespaces, "Namespaces to be excluded, split by commas. Example: --exclude-namespaces ns1,ns2,ns3. If --include-namespaces is set, --exclude-namespaces will be ignored.")
	cmd.PersistentFlags().StringSliceVarP(&opts.IncludeNamespaces, "include-namespaces", "n", opts.IncludeNamespaces, "Namespaces to run on, split by commas. Example: --include-namespaces ns1,ns2,ns3. If set, non-namespaced resources will be ignored.")
	rootCmd.PersistentFlags().StringSliceVar(&opts.IncludeThirdPartyCrds, "include-third-party-crds", opts.IncludeThirdPartyCrds, "Custom resources defintions to search, split by commas. Example: --include-crds argo-rollouts. If set, the chosen crd will be returned (in addition to the other resources, of course)")
}
