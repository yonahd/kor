package kor

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yonahd/kor/pkg/kor"
	"github.com/yonahd/kor/pkg/utils"
)

var allCmd = &cobra.Command{
	Use:   "all",
	Short: "Gets unused namespaced resources",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		clientset := kor.GetKubeClient(kubeconfig)
		apiExtClient := kor.GetAPIExtensionsClient(kubeconfig)
		dynamicClient := kor.GetDynamicClient(kubeconfig)
		kor.SetNamespacedFlagState(cmd.Flags().Changed("namespaced"))

		if response, err := kor.GetUnusedAll(filterOptions, clientset, apiExtClient, dynamicClient, outputFormat, opts); err != nil {
			fmt.Println(err)
		} else {
			utils.PrintLogo(outputFormat)
			fmt.Println(response)
		}
	},
}

func init() {
	allCmd.Flags().BoolVar(&opts.Namespaced, "namespaced", true, "If false, non-namespaced resources will be returned, otherwise returning namespaced resources by default. If not used, both are returned")
	rootCmd.AddCommand(allCmd)
}
