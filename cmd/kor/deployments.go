package kor

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var deployCmd = &cobra.Command{
	Use:     "deployment",
	Aliases: []string{"deploy", "deployments"},
	Short:   "Gets unused deployments",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		clientset := kor.GetKubeClient(kubeconfig)
		if response, err := kor.GetUnusedDeployments(includeExcludeLists, filterOptions, clientset, outputFormat, slackOpts); err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(response)
		}
	},
}

func init() {
	rootCmd.AddCommand(deployCmd)
}
