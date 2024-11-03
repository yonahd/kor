package kor

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yonahd/kor/pkg/clusterconfig"
	"github.com/yonahd/kor/pkg/kor"
	"github.com/yonahd/kor/pkg/utils"
)

var allCmd = &cobra.Command{
	Use:   "all",
	Short: "Gets unused resources",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		clientset := clusterconfig.GetKubeClient(kubeconfig)
		apiExtClient := clusterconfig.GetAPIExtensionsClient(kubeconfig)
		dynamicClient := clusterconfig.GetDynamicClient(kubeconfig)
		clientsetinterface, _ := clusterconfig.GetKubeClientForCrds(kubeconfig, clientset)
		if response, err := kor.GetUnusedAll(filterOptions, clientset, apiExtClient, dynamicClient, clientsetinterface, outputFormat, opts); err != nil {
			fmt.Println(err)
		} else {
			utils.PrintLogo(outputFormat)
			fmt.Println(response)
		}
	},
}

func init() {
	rootCmd.AddCommand(allCmd)
}
