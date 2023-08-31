package kor

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var pvcCmd = &cobra.Command{
	Use:   "pvc",
	Short: "Gets unused pvcs",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if outputFormat == "json" {
			if jsonResponse, err := kor.GetUnusedPvcsJson(namespace, kubeconfig); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(jsonResponse)
			}
		} else {
			kor.GetUnusedPvcs(namespace, kubeconfig)
		}
	},
}

func init() {
	rootCmd.AddCommand(pvcCmd)
}
