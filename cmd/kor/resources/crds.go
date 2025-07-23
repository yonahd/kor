package resources

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yonahd/kor/pkg/kor"
	"github.com/yonahd/kor/pkg/utils"
)

var crdCmd = &cobra.Command{
	Use:     "customresourcedefinition",
	Aliases: []string{"crd", "crds", "customresourcedefinitions"},
	Short:   "Gets unused crds",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		apiExtClient := kor.GetAPIExtensionsClient(Kubeconfig)
		dynamicClient := kor.GetDynamicClient(Kubeconfig)
		if response, err := kor.GetUnusedCrds(FilterOptions, apiExtClient, dynamicClient, OutputFormat, Opts); err != nil {
			fmt.Println(err)
		} else {
			utils.PrintLogo(korcmd.OutputFormat)
			fmt.Println(response)
		}

	},
}

func init() {
	korcmd.RootCmd.AddCommand(crdCmd)
}
