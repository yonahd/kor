package resources

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
		clientset := kor.GetKubeClient(korcmd.Kubeconfig)
		dynamicClient := kor.GetDynamicClient(Kubeconfig)

		if response, err := kor.GetUnusedfinalizers(FilterOptions, clientset, dynamicClient, OutputFormat, Opts); err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(response)
		}
	},
}

func init() {
	korcmd.RootCmd.AddCommand(finalizerCmd)
}
