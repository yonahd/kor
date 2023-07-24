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
		kor.GetUnusedStatefulsets(namespace)

	},
}

func init() {
	stsCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "", "Namespace to run on")
	rootCmd.AddCommand(stsCmd)
}
