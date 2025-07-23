package resources

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yonahd/kor/pkg/kor"
	"github.com/yonahd/kor/pkg/utils"
)

var pvCmd = &cobra.Command{
	Use:     "persistentvolume",
	Aliases: []string{"pv", "persistentvolumes"},
	Short:   "Gets unused pvs",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		clientset := kor.GetKubeClient(korcmd.Kubeconfig)

		if response, err := kor.GetUnusedPvs(korcmd.FilterOptions, clientset, korcmd.OutputFormat, korcmd.Opts); err != nil {
			fmt.Println(err)
		} else {
			utils.PrintLogo(korcmd.OutputFormat)
			fmt.Println(response)
		}

	},
}

func init() {
	korcmd.RootCmd.AddCommand(pvCmd)
}
