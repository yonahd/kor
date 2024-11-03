package kor

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yonahd/kor/pkg/clusterconfig"
	"github.com/yonahd/kor/pkg/kor"
	"github.com/yonahd/kor/pkg/utils"
)

var hpaCmd = &cobra.Command{
	Use:     "horizontalpodautoscaler",
	Aliases: []string{"hpa", "horizontalpodautoscalers"},
	Short:   "Gets unused hpas",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		clientset := clusterconfig.GetKubeClient(kubeconfig)

		if response, err := kor.GetUnusedHpas(filterOptions, clientset, outputFormat, opts); err != nil {
			fmt.Println(err)
		} else {
			utils.PrintLogo(outputFormat)
			fmt.Println(response)
		}

	},
}

func init() {
	rootCmd.AddCommand(hpaCmd)
}
