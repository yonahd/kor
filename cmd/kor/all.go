package kor

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yonahd/kor/pkg/kor"
	"github.com/yonahd/kor/pkg/utils"
)

var allCmd = &cobra.Command{
	Use:   "all",
	Short: "Gets unused resources",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		clientset := kor.GetKubeClient(Kubeconfig)
		apiExtClient := kor.GetAPIExtensionsClient(Kubeconfig)
		dynamicClient := kor.GetDynamicClient(Kubeconfig)
		kor.SetNamespacedFlagState(cmd.Flags().Changed("namespaced"))

		if response, err := kor.GetUnusedAll(FilterOptions, clientset, apiExtClient, dynamicClient, OutputFormat, Opts); err != nil {
			fmt.Println(err)
		} else {
			utils.PrintLogo(OutputFormat)
			fmt.Println(response)
		}
	},
}

func init() {
	allCmd.Flags().BoolVar(&Opts.Namespaced, "namespaced", true, "If false, non-namespaced resources will be returned, otherwise returning namespaced resources by default. If not used, both are returned")
	RootCmd.AddCommand(allCmd)
}
