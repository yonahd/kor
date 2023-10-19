package kor

import (
	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var exporterCmd = &cobra.Command{
	Use:   "exporter",
	Short: "start prometheus exporter",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		clientset := kor.GetKubeClient(kubeconfig)
		kor.Exporter(includeExcludeLists, clientset, "json", slackOpts)

	},
}

func init() {
	rootCmd.AddCommand(exporterCmd)
}
