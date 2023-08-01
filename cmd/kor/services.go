package kor

import (
	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var serviceCmd = &cobra.Command{
	Use:     "services",
	Aliases: []string{"svc"},
	Short:   "Gets unused services",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if outputFormat == "json" {
			kor.GetUnusedServicesJSON(namespace, kubeconfig)
		} else {
			kor.GetUnusedServices(namespace, kubeconfig)
		}

	},
}

func init() {
	serviceCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "Path to kubeconfig file (optional)")
	serviceCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "", "Namespace to run on")
	serviceCmd.PersistentFlags().StringVar(&outputFormat, "output", "table", "Output format (table or json)")
	rootCmd.AddCommand(serviceCmd)
}
