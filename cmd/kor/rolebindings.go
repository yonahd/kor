package kor

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yonahd/kor/pkg/kor"
	"github.com/yonahd/kor/pkg/utils"
)

var roleBindingCmd = &cobra.Command{
	Use:     "rolebinding",
	Aliases: []string{"rolebindings"},
	Short:   "Gets unused role bindings",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		clientset := kor.GetKubeClient(kubeConfig, kubeContext)

		if response, err := kor.GetUnusedRoleBindings(filterOptions, clientset, outputFormat, opts); err != nil {
			fmt.Println(err)
		} else {
			utils.PrintLogo(outputFormat)
			fmt.Println(response)
		}
	},
}

func init() {
	rootCmd.AddCommand(roleBindingCmd)
}
