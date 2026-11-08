package kor

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestShouldSkipKubeInitialization(t *testing.T) {
	cases := []struct {
		name       string
		cmdName    string
		parentName string
		expected   bool
	}{
		{name: "version command", cmdName: "version", expected: true},
		{name: "help command", cmdName: "help", expected: true},
		{name: "completion command", cmdName: "completion", expected: true},
		{name: "bash completion child", cmdName: "bash", parentName: "completion", expected: true},
		{name: "zsh completion child", cmdName: "zsh", parentName: "completion", expected: true},
		{name: "fish completion child", cmdName: "fish", parentName: "completion", expected: true},
		{name: "powershell completion child", cmdName: "powershell", parentName: "completion", expected: true},
		{name: "bash standalone", cmdName: "bash", expected: true},
		{name: "zsh standalone", cmdName: "zsh", expected: true},
		{name: "resource command", cmdName: "configmaps", parentName: "kor", expected: false},
		{name: "all command", cmdName: "all", expected: false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: c.cmdName}
			if c.parentName != "" {
				parent := &cobra.Command{Use: c.parentName}
				parent.AddCommand(cmd)
			}

			got := shouldSkipKubeInitialization(cmd)
			if got != c.expected {
				t.Errorf("Expected %v, got %v", c.expected, got)
			}
		})
	}
}

func TestExecName(t *testing.T) {
	if name := execName(); name != "kor" {
		t.Errorf("Expected kor, got %s", name)
	}
}
