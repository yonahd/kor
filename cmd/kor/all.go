package kor

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var allCmd = &cobra.Command{
	Use:   "all",
	Short: "Gets unused resources",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		clientset := kor.GetKubeClient(kubeconfig)

		if response, err := kor.GetUnusedAll(includeExcludeLists, clientset, outputFormat, opts); err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(response)
		}
	},
}

func init() {
	rootCmd.AddCommand(allCmd)
}
