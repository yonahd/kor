package kor

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var secretCmd = &cobra.Command{
	Use:     "secret",
	Aliases: []string{"scrt", "secrets"},
	Short:   "Gets unused secrets",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if outputFormat == "json" || outputFormat == "yaml" {
			if response, err := kor.GetUnusedSecretsStructured(includeExcludeLists, kubeconfig, outputFormat); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(response)
			}
		} else {
			kor.GetUnusedSecrets(includeExcludeLists, kubeconfig)
		}

	},
}

func init() {
	rootCmd.AddCommand(secretCmd)
}
