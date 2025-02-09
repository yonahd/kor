package kor

import (
	"github.com/spf13/cobra"

	"github.com/yonahd/kor/pkg/kor"
)

var resourceList []string

var exporterCmd = &cobra.Command{
	Use:   "exporter",
	Short: "start prometheus exporter",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		clientset := kor.GetKubeClient(kubeconfig)
		apiExtClient := kor.GetAPIExtensionsClient(kubeconfig)
		dynamicClient := kor.GetDynamicClient(kubeconfig)
		kor.SetNamespacedFlagState(cmd.Flags().Changed("namespaced"))
		
		kor.Exporter(filterOptions, clientset, apiExtClient, dynamicClient, "json", opts, resourceList)

	},
}

func init() {
	exporterCmd.Flags().StringSliceVarP(&resourceList, "resources", "r", nil, "Comma-separated list of resources to monitor (e.g., deployment,service)")
	exporterCmd.Flags().BoolVar(&opts.Namespaced, "namespaced", true, "If false, non-namespaced resources will be returned, otherwise returning namespaced resources by default. If not used, both are returned")
	rootCmd.AddCommand(exporterCmd)
}
