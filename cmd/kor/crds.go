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
		if response, err := kor.GetUnusedCrds(apiExtClient, dynamicClient, outputFormat, opts); err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(response)
		}

	},
}

func init() {
	rootCmd.AddCommand(crdCmd)
}
