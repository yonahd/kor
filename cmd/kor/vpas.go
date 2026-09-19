package kor

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yonahd/kor/pkg/kor"
	"github.com/yonahd/kor/pkg/utils"
)

var vpaCmd = &cobra.Command{
	Use:     "verticalpodautoscaler",
	Aliases: []string{"vpa", "verticalpodautoscalers"},
	Short:   "Gets unused vpas",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		clientset := kor.GetKubeClient(kubeconfig)
		dynamicClient := kor.GetDynamicClient(kubeconfig)

		if response, err := kor.GetUnusedVpas(filterOptions, clientset, dynamicClient, outputFormat, opts); err != nil {
			fmt.Println(err)
		} else {
			utils.PrintLogo(outputFormat, opts.ClusterName)
			fmt.Println(response)
		}
	},
}

func init() {
	rootCmd.AddCommand(vpaCmd)
}
