package kor

import (
	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var hpaCmd = &cobra.Command{
	Use:   "hpa",
	Short: "Gets unused hpas",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if outputFormat == "json" {
			kor.GetUnusedHpasJson(namespace, kubeconfig)
		} else {
			kor.GetUnusedHpas(namespace, kubeconfig)
		}
	},
}

func init() {
	hpaCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "Path to kubeconfig file (optional)")
	hpaCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "", "Namespace to run on")
	hpaCmd.PersistentFlags().StringVar(&outputFormat, "output", "table", "Output format (table or json)")
	rootCmd.AddCommand(hpaCmd)
}
