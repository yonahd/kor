package kor

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var roleCmd = &cobra.Command{
	Use:   "role",
	Short: "Gets unused roles",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if outputFormat == "json" {
			if jsonResponse, err := kor.GetUnusedRolesJSON(includeExcludeLists, kubeconfig); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(jsonResponse)
			}
		} else if outputFormat == "yaml" {
			if yamlResponse, err := kor.GetUnusedRolesYAML(includeExcludeLists, kubeconfig); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(yamlResponse)
			}
		} else {
			kor.GetUnusedRoles(includeExcludeLists, kubeconfig)
		}
	},
}

func init() {
	rootCmd.AddCommand(roleCmd)
}
