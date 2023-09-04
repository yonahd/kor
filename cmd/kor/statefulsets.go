package kor

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var stsCmd = &cobra.Command{
	Use:     "statefulsets",
	Aliases: []string{"sts"},
	Short:   "Gets unused statefulsets",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if outputFormat == "yaml" {
			if yamlResponse, err := kor.GetUnusedStatefulsetsYAML(includeExcludeLists, kubeconfig); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(yamlResponse)
			}
		} else {
			kor.GetUnusedStatefulsets(includeExcludeLists, kubeconfig)
		}

	},
}

func init() {
	rootCmd.AddCommand(stsCmd)
}
