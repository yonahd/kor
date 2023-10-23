package kor

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var pdbCmd = &cobra.Command{
	Use:     "poddisruptionbudget",
	Aliases: []string{"pdb", "poddisruptionbudgets"},
	Short:   "Gets unused pdbs",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		clientset := kor.GetKubeClient(kubeconfig)

		if response, err := kor.GetUnusedPdbs(includeExcludeLists, filterOptions, clientset, outputFormat, slackOpts); err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(response)
		}
	},
}

func init() {
	rootCmd.AddCommand(pdbCmd)
}
