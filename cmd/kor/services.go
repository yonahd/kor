package kor

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yonahd/kor/pkg/kor"
	"github.com/yonahd/kor/pkg/utils"
)

var serviceCmd = &cobra.Command{
	Use:     "service",
	Aliases: []string{"svc", "services"},
	Short:   "Gets unused services",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		cluster := kor.GetClusterName(kubeconfig)
		clientset := kor.GetKubeClient(kubeconfig)

		if response, err := kor.GetUnusedServices(filterOptions, clientset, outputFormat, opts); err != nil {
			fmt.Println(err)
		} else {
			utils.PrintLogo(outputFormat, cluster)
			fmt.Println(response)
		}
	},
}

func init() {
	rootCmd.AddCommand(serviceCmd)
}
