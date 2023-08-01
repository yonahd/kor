package kor

import (
	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var deployCmd = &cobra.Command{
	Use:     "deployments",
	Aliases: []string{"deploy"},
	Short:   "Gets unused deployments",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if outputFormat == "json" {
			kor.GetUnusedDeploymentsJSON(namespace, kubeconfig)
		} else {
			kor.GetUnusedDeployments(namespace, kubeconfig)
		}

	},
}

func init() {
	deployCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "Path to kubeconfig file (optional)")
	deployCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "", "Namespace to run on")
	deployCmd.PersistentFlags().StringVar(&outputFormat, "output", "table", "Output format (table or json)")
	rootCmd.AddCommand(deployCmd)
}
