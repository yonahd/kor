package kor

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var pvcCmd = &cobra.Command{
	Use:     "persistentvolumeclaim",
	Aliases: []string{"pvc", "persistentvolumeclaims"},
	Short:   "Gets unused pvcs",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		clientset := kor.GetKubeClient(kubeconfig)

		if response, err := kor.GetUnusedPvcs(includeExcludeLists, clientset, outputFormat, slackOpts, deleteOpts); err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(response)
		}

	},
}

func init() {
	rootCmd.AddCommand(pvcCmd)
}
