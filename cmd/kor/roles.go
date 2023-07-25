package kor

import (
	"github.com/spf13/cobra"
	"github.com/yonahd/kor/pkg/kor"
)

var roleCmd = &cobra.Command{
	Use:   "role",
	Short: "Gets unused roles",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		kor.GetUnusedRoles(namespace)

	},
}

func init() {
	roleCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "", "Namespace to run on")
	rootCmd.AddCommand(roleCmd)
}
