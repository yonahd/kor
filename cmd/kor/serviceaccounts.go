package kor

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yonahd/kor/pkg/clusterconfig"
	"github.com/yonahd/kor/pkg/kor"
	"github.com/yonahd/kor/pkg/utils"
)

var serviceAccountCmd = &cobra.Command{
	Use:     "serviceaccount",
	Aliases: []string{"sa", "serviceaccounts"},
	Short:   "Gets unused service accounts",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		clientset := clusterconfig.GetKubeClient(kubeconfig)

		if response, err := kor.GetUnusedServiceAccounts(filterOptions, clientset, outputFormat, opts); err != nil {
			fmt.Println(err)
		} else {
			utils.PrintLogo(outputFormat)
			fmt.Println(response)
		}
	},
}

func init() {
	rootCmd.AddCommand(serviceAccountCmd)
}
