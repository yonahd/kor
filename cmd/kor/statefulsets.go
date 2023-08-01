package kor

import (
	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var stsCmd = &cobra.Command{
	Use:     "statefulsets",
	Aliases: []string{"sts"},
	Short:   "Gets unused statefulsets",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if outputFormat == "json" {
			kor.GetUnusedStatefulsetsJSON(namespace, kubeconfig)
		} else {
			kor.GetUnusedStatefulsets(namespace, kubeconfig)
		}

	},
}

func init() {
	stsCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "Path to kubeconfig file (optional)")
	stsCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "", "Namespace to run on")
	stsCmd.PersistentFlags().StringVar(&outputFormat, "output", "table", "Output format (table or json)")
	rootCmd.AddCommand(stsCmd)
}
