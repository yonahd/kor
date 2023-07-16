package kor

import (
	"github.com/spf13/cobra"
	"github.com/ydissen/kor/pkg/kor"
)

var configmapCmd = &cobra.Command{
	Use:     "configmap",
	Aliases: []string{"cm"},
	Short:   "Gets unused configmaps",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		kor.GetUnusedConfigmaps(namespace)

	},
}

func init() {
	configmapCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "", "Namespace to run on")
	rootCmd.AddCommand(configmapCmd)
}
