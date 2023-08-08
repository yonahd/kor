package kor

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kor",
	Short: "kor - a CLI to to discover unused Kubernetes resources",
	Long: `kor is a CLI to to discover unused Kubernetes resources 
	kor can currently discover unused configmaps and secrets`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

	},
}

var namespace string
var outputFormat string
var kubeconfig string

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error while executing your CLI '%s'", err)
		os.Exit(1)
	}
}
