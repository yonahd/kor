package kor

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/yonahd/kor/pkg/kor"
	"github.com/yonahd/kor/pkg/utils"
)

var volumeAttachmentCmd = &cobra.Command{
	Use:     "volumeattachment",
	Aliases: []string{"volumeattachments"},
	Short:   "Gets unused volumeattachments",
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cluster := kor.GetClusterName(kubeconfig)
		clientset := kor.GetKubeClient(kubeconfig)

		if response, err := kor.GetUnusedVolumeAttachments(filterOptions, clientset, outputFormat, opts); err != nil {
			fmt.Println(err)
		} else {
			utils.PrintLogo(outputFormat, cluster)
			fmt.Println(response)
		}
	},
}

func init() {
	rootCmd.AddCommand(volumeAttachmentCmd)
}
