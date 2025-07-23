package resources

import (
	"fmt"

	"github.com/spf13/cobra"
	korcmd "github.com/yonahd/kor/cmd/kor"

	"github.com/yonahd/kor/pkg/kor"
	"github.com/yonahd/kor/pkg/utils"
)

var dsCmd = &cobra.Command{
	Use:     "daemonset",
	Aliases: []string{"ds", "daemonsets"},
	Short:   "Gets unused daemonSets",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		clientset := kor.GetKubeClient(korcmd.Kubeconfig)

		if response, err := kor.GetUnusedDaemonSets(korcmd.FilterOptions, clientset, korcmd.OutputFormat, korcmd.Opts); err != nil {
			fmt.Println(err)
		} else {
			utils.PrintLogo(korcmd.OutputFormat)
			fmt.Println(response)
		}
	},
}

func init() {
	korcmd.RootCmd.AddCommand(dsCmd)
}
