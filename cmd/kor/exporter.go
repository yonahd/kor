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
		clientset := kor.GetKubeClient(kubeConfig, kubeContext)
		apiExtClient := kor.GetAPIExtensionsClient(kubeConfig)
		dynamicClient := kor.GetDynamicClient(kubeConfig)

		kor.Exporter(filterOptions, clientset, apiExtClient, dynamicClient, "json", opts, resourceList)

	},
}

func init() {
	exporterCmd.Flags().StringSliceVarP(&resourceList, "resources", "r", nil, "Comma-separated list of resources to monitor (e.g., deployment,service)")
	rootCmd.AddCommand(exporterCmd)
}
