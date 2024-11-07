package kor

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yonahd/kor/pkg/kor"
)

var finalizerCmd = &cobra.Command{
	Use:     "finalizer",
	Aliases: []string{"fin", "finalizers"},
	Short:   "Gets resources waiting for finalizers to delete",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		clientset := kor.GetKubeClient(kubeConfig, kubeContext)
		dynamicClient := kor.GetDynamicClient(kubeConfig)

		if response, err := kor.GetUnusedfinalizers(filterOptions, clientset, dynamicClient, outputFormat, opts); err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(response)
		}
	},
}

func init() {
	rootCmd.AddCommand(finalizerCmd)
}
