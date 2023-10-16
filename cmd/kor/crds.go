package kor

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var crdCmd = &cobra.Command{
	Use:     "customresourcedefinition",
	Aliases: []string{"crd", "customresourcedefinitions"},
	Short:   "Gets unused crds",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		apiExtClient := kor.GetAPIExtensionsClient(kubeconfig)
		dynamicClient := kor.GetDynamicClient(kubeconfig)
		if outputFormat == "json" || outputFormat == "yaml" {
			if response, err := kor.GetUnusedCrdsStructured(apiExtClient, dynamicClient, outputFormat); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(response)
			}
		} else {
			kor.GetUnusedCrds(apiExtClient, dynamicClient, slackOpts)
		}
	},
}

func init() {
	rootCmd.AddCommand(crdCmd)
}
