package kor

import (
	"github.com/spf13/cobra"

	"github.com/yonahd/kor/pkg/kor"
	"github.com/yonahd/kor/pkg/utils"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print kor version information",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		cluster := kor.GetClusterName(kubeconfig)
		utils.PrintVersion(cluster)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
