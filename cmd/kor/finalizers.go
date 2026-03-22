package kor

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yonahd/kor/pkg/kor"
	"github.com/yonahd/kor/pkg/utils"
)

var finalizerCmd = &cobra.Command{
	Use:     "finalizer",
	Aliases: []string{"fin", "finalizers"},
	Short:   "Gets resources waiting for finalizers to delete",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cluster := kor.GetClusterName(kubeconfig)
		clientset := kor.GetKubeClient(kubeconfig)
		dynamicClient := kor.GetDynamicClient(kubeconfig)

		if response, err := kor.GetUnusedfinalizers(filterOptions, clientset, dynamicClient, outputFormat, opts); err != nil {
			fmt.Println(err)
		} else {
			utils.PrintLogo(outputFormat, cluster)
			fmt.Println(response)
		}
	},
}

func init() {
	rootCmd.AddCommand(finalizerCmd)
}
