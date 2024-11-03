package kor

import (
	"github.com/spf13/cobra"

	"github.com/yonahd/kor/pkg/clusterconfig"
	"github.com/yonahd/kor/pkg/kor"
)

var resourceList []string

var exporterCmd = &cobra.Command{
	Use:   "exporter",
	Short: "start prometheus exporter",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		clientset := clusterconfig.GetKubeClient(kubeconfig)
		clientsetinterface, _ := clusterconfig.GetKubeClientForCrds(kubeconfig, clientset)
		apiExtClient := clusterconfig.GetAPIExtensionsClient(kubeconfig)
		dynamicClient := clusterconfig.GetDynamicClient(kubeconfig)

		kor.Exporter(filterOptions, clientset, apiExtClient, dynamicClient, clientsetinterface, "json", opts, resourceList)

	},
}

func init() {
	exporterCmd.Flags().StringSliceVarP(&resourceList, "resources", "r", nil, "Comma-separated list of resources to monitor (e.g., deployment,service)")
	rootCmd.AddCommand(exporterCmd)
}
