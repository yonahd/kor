package kor

import (
	"bytes"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
	"sort"
)

func RemoveDuplicatesAndSort(slice []string) []string {
	uniqueSet := make(map[string]bool)
	for _, item := range slice {
		uniqueSet[item] = true
	}
	uniqueSlice := make([]string, 0, len(uniqueSet))
	for item := range uniqueSet {
		uniqueSlice = append(uniqueSlice, item)
	}
	sort.Strings(uniqueSlice)
	return uniqueSlice
}

func GetKubeConfigPath() string {
	home := homedir.HomeDir()
	return filepath.Join(home, ".kube", "config")
}

func FormatOutput(namespace string, resources []string, resourceType string) string {
	if len(resources) == 0 {
		return fmt.Sprintf("No unused %s found in the namespace: %s", resourceType, namespace)
	}

	var buf bytes.Buffer
	table := tablewriter.NewWriter(&buf)
	table.SetHeader([]string{"#", "Resource Name"})

	for i, name := range resources {
		table.Append([]string{fmt.Sprintf("%d", i+1), name})
	}

	table.Render()

	return fmt.Sprintf("Unused %s in Namespace: %s\n%s", resourceType, namespace, buf.String())
}
