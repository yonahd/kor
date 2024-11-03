package kor

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yonahd/kor/pkg/clusterconfig"
	"github.com/yonahd/kor/pkg/kor"
	"github.com/yonahd/kor/pkg/utils"
)

var clusterRoleCmd = &cobra.Command{
	Use:     "clusterrole",
	Aliases: []string{"clusterroles"},
	Short:   "Gets unused cluster roles",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		clientset := clusterconfig.GetKubeClient(kubeconfig)

		if response, err := kor.GetUnusedClusterRoles(filterOptions, clientset, outputFormat, opts); err != nil {
			fmt.Println(err)
		} else {
			utils.PrintLogo(outputFormat)
			fmt.Println(response)
		}
	},
}

func init() {
	rootCmd.AddCommand(clusterRoleCmd)
}
