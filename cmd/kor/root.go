package kor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

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

var RootCmd = &cobra.Command{
	Use:   execName(),
	Short: "kor - a CLI to discover unused Kubernetes resources",
	Long: `kor is a CLI to discover unused Kubernetes resources
	kor can currently discover unused configmaps and secrets`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		resourceNames := args[0]
		clientset := kor.GetKubeClient(Kubeconfig)
		apiExtClient := kor.GetAPIExtensionsClient(Kubeconfig)
		dynamicClient := kor.GetDynamicClient(Kubeconfig)

		if response, err := kor.GetUnusedMulti(resourceNames, FilterOptions, clientset, apiExtClient, dynamicClient, OutputFormat, Opts); err != nil {
			fmt.Println(err)
		} else {
			utils.PrintLogo(OutputFormat)
			fmt.Println(response)
		}
	},
}

var (
	OutputFormat  string
	Kubeconfig    string
	Opts          common.Opts
	FilterOptions = &filters.Options{}
)

func init() {
	initFlags()
	initViper()
	initKindsList()
	addFilterOptionsFlag(RootCmd, FilterOptions)
}

func initKindsList() {
	clientset := kor.GetKubeClient(Kubeconfig)
	kor.ResourceKindList, _ = kor.GetResourceKinds(clientset)
}

func initFlags() {
	RootCmd.PersistentFlags().StringVarP(&Kubeconfig, "kubeconfig", "k", "", "Path to kubeconfig file (optional)")
	RootCmd.PersistentFlags().StringVarP(&OutputFormat, "output", "o", "table", "Output format (table, json or yaml)")
	RootCmd.PersistentFlags().StringVar(&Opts.WebhookURL, "slack-webhook-url", "", "Slack webhook URL to send notifications to")
	RootCmd.PersistentFlags().StringVar(&Opts.Channel, "slack-channel", "", "Slack channel to send notifications to, requires --slack-auth-token to be set")
	RootCmd.PersistentFlags().StringVar(&Opts.Token, "slack-auth-token", "", "Slack auth token to send notifications to, requires --slack-channel to be set")
	RootCmd.PersistentFlags().BoolVar(&Opts.DeleteFlag, "delete", false, "Delete unused resources")
	RootCmd.PersistentFlags().BoolVar(&Opts.NoInteractive, "no-interactive", false, "Do not prompt for confirmation when deleting resources. Be careful when using this flag!")
	RootCmd.PersistentFlags().BoolVarP(&Opts.Verbose, "verbose", "v", false, "Verbose output (print empty namespaces)")
	RootCmd.PersistentFlags().StringVar(&Opts.GroupBy, "group-by", "namespace", "Group output by (namespace, resource)")
	RootCmd.PersistentFlags().BoolVar(&Opts.ShowReason, "show-reason", false, "Print reason resource is considered unused")
	RootCmd.PersistentFlags().BoolVar(&Opts.ShowReason, "include-crds", false, "Find unused CRDs (Custom Resource Definitions) as well.")
}

func initViper() {
	if err := viper.BindEnv("slack-webhook-url", "SLACK_WEBHOOK_URL"); err != nil {
		fmt.Printf("Error binding SLACK_WEBHOOK_URL: %v\n", err)
	}
	if err := viper.BindEnv("slack-auth-token", "SLACK_AUTH_TOKEN"); err != nil {
		fmt.Printf("Error binding SLACK_AUTH_TOKEN: %v\n", err)
	}

	viper.AutomaticEnv()

	if err := viper.BindPFlag("slack-webhook-url", RootCmd.PersistentFlags().Lookup("slack-webhook-url")); err != nil {
		fmt.Printf("Error binding flag --slack-webhook-url: %v\n", err)
	}

	if err := viper.BindPFlag("slack-auth-token", RootCmd.PersistentFlags().Lookup("slack-auth-token")); err != nil {
		fmt.Printf("Error binding flag --slack-auth-token: %v\n", err)
	}

	Opts.WebhookURL = viper.GetString("slack-webhook-url")
	Opts.Token = viper.GetString("slack-auth-token")

	Opts.WebhookURL = os.ExpandEnv(Opts.WebhookURL)
	Opts.Token = os.ExpandEnv(Opts.Token)
}

func Execute() {
	_ = RootCmd.ParseFlags(os.Args)
	if err := FilterOptions.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Error while validating filter options '%s'", err)
		os.Exit(1)
	}
	FilterOptions.Modify()
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error while executing your CLI '%s'", err)
		os.Exit(1)
	}
}

func addFilterOptionsFlag(cmd *cobra.Command, opts *filters.Options) {
	cmd.PersistentFlags().StringSliceVarP(&opts.ExcludeLabels, "exclude-labels", "l", opts.ExcludeLabels, "Selector to filter out, Example: --exclude-labels key1=value1,key2=value2. If --include-labels is set, --exclude-labels will be ignored")
	cmd.PersistentFlags().StringVar(&opts.NewerThan, "newer-than", opts.NewerThan, "The maximum age of the resources to be considered unused. This flag cannot be used together with older-than flag. Example: --newer-than=1h2m")
	cmd.PersistentFlags().StringVar(&opts.OlderThan, "older-than", opts.OlderThan, "The minimum age of the resources to be considered unused. This flag cannot be used together with newer-than flag. Example: --older-than=1h2m")
	cmd.PersistentFlags().StringVar(&opts.IncludeLabels, "include-labels", opts.IncludeLabels, "Selector to filter in, Example: --include-labels key1=value1 (currently supports one label)")
	cmd.PersistentFlags().StringSliceVarP(&opts.ExcludeNamespaces, "exclude-namespaces", "e", opts.ExcludeNamespaces, "Namespaces to be excluded, split by commas. Example: --exclude-namespaces ns1,ns2,ns3. If --include-namespaces is set, --exclude-namespaces will be ignored")
	cmd.PersistentFlags().StringSliceVarP(&opts.IncludeNamespaces, "include-namespaces", "n", opts.IncludeNamespaces, "Namespaces to run on, split by commas. Example: --include-namespaces ns1,ns2,ns3. If set, non-namespaced resources will be ignored")
}
