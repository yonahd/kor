package kor

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var configmapCmd = &cobra.Command{
	Use:     "configmap",
	Aliases: []string{"cm", "configmaps"},
	Short:   "Gets unused configmaps",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		clientset := kor.GetKubeClient(kubeconfig)
		if response, err := kor.GetUnusedConfigmaps(includeExcludeLists, filterOptions, clientset, outputFormat, slackOpts); err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(response)
		}
	},
}

func init() {
	rootCmd.AddCommand(configmapCmd)
}
