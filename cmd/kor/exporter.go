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
		clientset := kor.GetKubeClient(Kubeconfig)
		apiExtClient := kor.GetAPIExtensionsClient(Kubeconfig)
		dynamicClient := kor.GetDynamicClient(Kubeconfig)
		kor.SetNamespacedFlagState(cmd.Flags().Changed("namespaced"))
		kor.Exporter(FilterOptions, clientset, apiExtClient, dynamicClient, "json", Opts, resourceList)

	},
}

func init() {
	exporterCmd.Flags().StringSliceVarP(&resourceList, "resources", "r", nil, "Comma-separated list of resources to monitor (e.g., deployment,service)")
	exporterCmd.Flags().BoolVar(&Opts.Namespaced, "namespaced", true, "If false, non-namespaced resources will be returned, otherwise returning namespaced resources by default. If not used, both are returned")
	RootCmd.AddCommand(exporterCmd)
}
