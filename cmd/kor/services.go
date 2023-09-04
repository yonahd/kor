package kor

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var serviceCmd = &cobra.Command{
	Use:     "services",
	Aliases: []string{"svc"},
	Short:   "Gets unused services",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if outputFormat == "json" {
			if jsonResponse, err := kor.GetUnusedServicesJSON(includeExcludeLists, kubeconfig); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(jsonResponse)
			}
		} else if outputFormat == "yaml" {
			if yamlResponse, err := kor.GetUnusedServicesYAML(includeExcludeLists, kubeconfig); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(yamlResponse)
			}
		} else {
			kor.GetUnusedServices(includeExcludeLists, kubeconfig)
		}

	},
}

func init() {
	rootCmd.AddCommand(serviceCmd)
}
