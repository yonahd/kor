package resources

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yonahd/kor/pkg/kor"
	"github.com/yonahd/kor/pkg/utils"
)

var jobCmd = &cobra.Command{
	Use:     "job",
	Aliases: []string{"jobs"},
	Short:   "Gets unused jobs",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		clientset := kor.GetKubeClient(korcmd.Kubeconfig)

		if response, err := kor.GetUnusedJobs(korcmd.FilterOptions, clientset, korcmd.OutputFormat, korcmd.Opts); err != nil {
			fmt.Println(err)
		} else {
			utils.PrintLogo(korcmd.OutputFormat)
			fmt.Println(response)
		}
	},
}

func init() {
	korcmd.RootCmd.AddCommand(jobCmd)
}
