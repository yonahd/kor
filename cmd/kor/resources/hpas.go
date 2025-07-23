package resources

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yonahd/kor/pkg/kor"
	"github.com/yonahd/kor/pkg/utils"
)

var hpaCmd = &cobra.Command{
	Use:     "horizontalpodautoscaler",
	Aliases: []string{"hpa", "horizontalpodautoscalers"},
	Short:   "Gets unused hpas",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		clientset := kor.GetKubeClient(korcmd.Kubeconfig)

		if response, err := kor.GetUnusedHpas(korcmd.FilterOptions, clientset, korcmd.OutputFormat, korcmd.Opts); err != nil {
			fmt.Println(err)
		} else {
			utils.PrintLogo(korcmd.OutputFormat)
			fmt.Println(response)
		}

	},
}

func init() {
	korcmd.RootCmd.AddCommand(hpaCmd)
}
