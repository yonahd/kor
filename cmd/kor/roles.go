package kor

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var roleCmd = &cobra.Command{
	Use:     "role",
	Aliases: []string{"roles"},
	Short:   "Gets unused roles",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		clientset := kor.GetKubeClient(kubeconfig)
		if outputFormat == "json" || outputFormat == "yaml" {
			if response, err := kor.GetUnusedRolesStructured(includeExcludeLists, clientset, outputFormat); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(response)
			}
		} else {
			kor.GetUnusedRoles(includeExcludeLists, clientset, slackOpts)
		}

	},
}

func init() {
	rootCmd.AddCommand(roleCmd)
}
