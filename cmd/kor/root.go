package kor

import (
	"fmt"
	"os"

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
		if outputFormat == "json" || outputFormat == "yaml" {
			if response, err := kor.GetUnusedMultiStructured(includeExcludeLists, kubeconfig, outputFormat, resourceNames); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(response)
			}
		} else {
			kor.GetUnusedMulti(includeExcludeLists, kubeconfig, resourceNames)
		}
	},
}

var (
	outputFormat        string
	kubeconfig          string
	includeExcludeLists kor.IncludeExcludeLists
)

func Execute() {
	rootCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "Path to kubeconfig file (optional)")
	rootCmd.PersistentFlags().StringVarP(&includeExcludeLists.IncludeListStr, "include-namespaces", "n", "", "Namespaces to run on, splited by comma. Example: --include-namespace ns1,ns2,ns3. ")
	rootCmd.PersistentFlags().StringVarP(&includeExcludeLists.ExcludeListStr, "exclude-namespaces", "e", "", "Namespaces to be excluded, splited by comma. Example: --exclude-namespace ns1,ns2,ns3. If --include-namespace is set, --exclude-namespaces will be ignored.")
	rootCmd.PersistentFlags().StringVar(&outputFormat, "output", "table", "Output format (table or json)")
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error while executing your CLI '%s'", err)
		os.Exit(1)
	}
}
