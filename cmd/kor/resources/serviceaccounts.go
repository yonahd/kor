package resources

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yonahd/kor/pkg/kor"
	"github.com/yonahd/kor/pkg/utils"
)

var serviceAccountCmd = &cobra.Command{
	Use:     "serviceaccount",
	Aliases: []string{"sa", "serviceaccounts"},
	Short:   "Gets unused service accounts",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		clientset := kor.GetKubeClient(korcmd.Kubeconfig)

		if response, err := kor.GetUnusedServiceAccounts(korcmd.FilterOptions, clientset, korcmd.OutputFormat, korcmd.Opts); err != nil {
			fmt.Println(err)
		} else {
			utils.PrintLogo(korcmd.OutputFormat)
			fmt.Println(response)
		}
	},
}

func init() {
	korcmd.RootCmd.AddCommand(serviceAccountCmd)
}
