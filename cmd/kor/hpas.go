package kor

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var hpaCmd = &cobra.Command{
	Use:     "horizontalpodautoscaler",
	Aliases: []string{"hpa", "horizontalpodautoscalers"},
	Short:   "Gets unused hpas",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		clientset := kor.GetKubeClient(kubeconfig)

		if response, err := kor.GetUnusedHpas(includeExcludeLists, clientset, outputFormat, opts); err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(response)
		}

	},
}

func init() {
	rootCmd.AddCommand(hpaCmd)
}
