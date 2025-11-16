package kor

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yonahd/kor/pkg/kor"
	"github.com/yonahd/kor/pkg/utils"
)

var priorityClassCmd = &cobra.Command{
	Use:     "priorityclass",
	Aliases: []string{"pc", "priorityclasses"},
	Short:   "Gets unused priorityClasses",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		clientset := kor.GetKubeClient(kubeconfig)

		if response, err := kor.GetUnusedPriorityClasses(filterOptions, clientset, outputFormat, opts); err != nil {
			fmt.Println(err)
		} else {
			utils.PrintLogo(outputFormat)
			fmt.Println(response)
		}

	},
}

func init() {
	rootCmd.AddCommand(priorityClassCmd)
}
