package kor

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var pdbCmd = &cobra.Command{
	Use:   "pdb",
	Short: "Gets unused pdbs",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if outputFormat == "json" || outputFormat == "yaml" {
			if response, err := kor.GetUnusedPdbsStructured(includeExcludeLists, kubeconfig, outputFormat); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(response)
			}
		} else {
			kor.GetUnusedPdbs(includeExcludeLists, kubeconfig)
		}

	},
}

func init() {
	rootCmd.AddCommand(pdbCmd)
}
