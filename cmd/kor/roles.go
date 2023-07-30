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
		if outputFormat == "json" {
			kor.GetUnusedRolesJSON(namespace)
		} else {
			kor.GetUnusedRoles(namespace)
		}
	},
}

func init() {
	roleCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "", "Namespace to run on")
	roleCmd.PersistentFlags().StringVar(&outputFormat, "output", "table", "Output format (table or json)")
	rootCmd.AddCommand(roleCmd)
}
