package kor

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yonahd/kor/pkg/clusterconfig"
	"github.com/yonahd/kor/pkg/kor"
	"github.com/yonahd/kor/pkg/utils"
)

var crdCmd = &cobra.Command{
	Use:     "customresourcedefinition",
	Aliases: []string{"crd", "crds", "customresourcedefinitions"},
	Short:   "Gets unused crds",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		apiExtClient := clusterconfig.GetAPIExtensionsClient(kubeconfig)
		dynamicClient := clusterconfig.GetDynamicClient(kubeconfig)
		if response, err := kor.GetUnusedCrds(filterOptions, apiExtClient, dynamicClient, outputFormat, opts); err != nil {
			fmt.Println(err)
		} else {
			utils.PrintLogo(outputFormat)
			fmt.Println(response)
		}

	},
}

func init() {
	rootCmd.AddCommand(crdCmd)
}
