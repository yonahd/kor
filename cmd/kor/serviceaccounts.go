package kor

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var serviceAccountCmd = &cobra.Command{
	Use:     "serviceaccount",
	Aliases: []string{"sa"},
	Short:   "Gets unused service accounts",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if outputFormat == "json" {
			if jsonResponse, err := kor.GetUnusedServiceAccountsJSON(includeExcludeLists, kubeconfig); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(jsonResponse)
			}
		} else if outputFormat == "yaml" {
			if yamlResponse, err := kor.GetUnusedServiceAccountsYAML(includeExcludeLists, kubeconfig); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(yamlResponse)
			}
		} else {
			kor.GetUnusedServiceAccounts(includeExcludeLists, kubeconfig)
		}

	},
}

func init() {
	rootCmd.AddCommand(serviceAccountCmd)
}
