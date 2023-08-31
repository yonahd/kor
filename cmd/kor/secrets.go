package kor

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var secretCmd = &cobra.Command{
	Use:     "secret",
	Aliases: []string{"scrt"},
	Short:   "Gets unused secrets",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if outputFormat == "json" {
			if jsonResponse, err := kor.GetUnusedSecretsJSON(namespace, kubeconfig); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(jsonResponse)
			}
		} else {
			kor.GetUnusedSecrets(namespace, kubeconfig)
		}

	},
}

func init() {
	rootCmd.AddCommand(secretCmd)
}
