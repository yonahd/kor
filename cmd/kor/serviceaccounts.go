package kor

import (
	"github.com/spf13/cobra"
	"github.com/ydissen/kor/pkg/kor"
)

var serviceAccountCmd = &cobra.Command{
	Use:     "serviceaccount",
	Aliases: []string{"sa"},
	Short:   "Gets unused service accounts",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		kor.GetUnusedServiceAccounts(namespace)

	},
}

func init() {
	serviceAccountCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "", "Namespace to run on")
	rootCmd.AddCommand(serviceAccountCmd)
}
