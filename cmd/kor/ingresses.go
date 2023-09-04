package kor

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var ingressCmd = &cobra.Command{
	Use:     "ingress",
	Aliases: []string{"ing"},
	Short:   "Gets unused ingresses",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if outputFormat == "json" {
			if jsonResponse, err := kor.GetUnusedIngressesJSON(includeExcludeLists, kubeconfig); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(jsonResponse)
			}
		} else if outputFormat == "yaml" {
			if yamlResponse, err := kor.GetUnusedIngressesYAML(includeExcludeLists, kubeconfig); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(yamlResponse)
			}
		} else {
			kor.GetUnusedIngresses(includeExcludeLists, kubeconfig)
		}

	},
}

func init() {
	rootCmd.AddCommand(ingressCmd)
}
