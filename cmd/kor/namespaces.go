package kor

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yonahd/kor/pkg/kor"
	"github.com/yonahd/kor/pkg/utils"
)

var namespaceCmd = &cobra.Command{
	Use:     "namespace",
	Aliases: []string{"ns", "namespaces"},
	Short:   "Gets unused namespaces",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		clientset := kor.GetKubeClient(kubeconfig)
		dynamicClient := kor.GetDynamicClient(kubeconfig)
		dicoveryClient := kor.GetDiscoveryClient(kubeconfig)

		if response, err := kor.GetUnusedNamespaces(ctx, filterOptions, clientset, dynamicClient, dicoveryClient, outputFormat, opts); err != nil {
			fmt.Println(err)
		} else {
			utils.PrintLogo(outputFormat)
			fmt.Println(response)
		}
	},
}

func init() {
	namespaceCmd.PersistentFlags().StringSliceVarP(
		&filterOptions.IgnoreResourceTypes,
		"ignore-resource-types",
		"i",
		filterOptions.IgnoreResourceTypes,
		"Child resource type selector to filter out from namespace emptiness evaluation,"+
			" example: --ignore-resource-types secrets,configmaps."+
			" Types should be specified in a format printed out in NAME column by 'kubectl api-resources --namespaced=true'.",
	)
	rootCmd.AddCommand(namespaceCmd)
}
