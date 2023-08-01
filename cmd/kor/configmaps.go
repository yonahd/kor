package kor

import (
	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var configmapCmd = &cobra.Command{
	Use:     "configmap",
	Aliases: []string{"cm"},
	Short:   "Gets unused configmaps",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if outputFormat == "json" {
			kor.GetUnusedConfigmapsJSON(namespace, kubeconfig)
		} else {
			kor.GetUnusedConfigmaps(namespace, kubeconfig)
		}

	},
}

func init() {
	configmapCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "Path to kubeconfig file (optional)")
	configmapCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "", "Namespace to run on")
	configmapCmd.PersistentFlags().StringVar(&outputFormat, "output", "table", "Output format (table or json)")
	rootCmd.AddCommand(configmapCmd)
}
