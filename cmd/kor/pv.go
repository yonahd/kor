package kor

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yonahd/kor/pkg/clusterconfig"
	"github.com/yonahd/kor/pkg/kor"
	"github.com/yonahd/kor/pkg/utils"
)

var pvCmd = &cobra.Command{
	Use:     "persistentvolume",
	Aliases: []string{"pv", "persistentvolumes"},
	Short:   "Gets unused pvs",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		clientset := clusterconfig.GetKubeClient(kubeconfig)

		if response, err := kor.GetUnusedPvs(filterOptions, clientset, outputFormat, opts); err != nil {
			fmt.Println(err)
		} else {
			utils.PrintLogo(outputFormat)
			fmt.Println(response)
		}

	},
}

func init() {
	rootCmd.AddCommand(pvCmd)
}
