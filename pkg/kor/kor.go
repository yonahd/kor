package kor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/olekukonko/tablewriter"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type ExceptionResource struct {
	ResourceName string
	Namespace    string
}
type IncludeExcludeLists struct {
	IncludeListStr string
	ExcludeListStr string
}

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

func GetKubeClient(kubeconfig string) *kubernetes.Clientset {
	if kubeconfig == "" {
		kubeconfig = GetKubeConfigPath()
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load kubeconfig: %v\n", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create Kubernetes client: %v\n", err)
		os.Exit(1)
	}
	return clientset
}

func SetNamespaceList(namespaceLists IncludeExcludeLists, kubeClient *kubernetes.Clientset) []string {
	namespaces := make([]string, 0)
	namespacesMap := make(map[string]bool)
	if namespaceLists.IncludeListStr != "" && namespaceLists.ExcludeListStr != "" {
		fmt.Fprintf(os.Stderr, "Exclude namespaces can't be used together with include namespaces. Ignoring --exclude-namespace(-e) flag\n")
		namespaceLists.ExcludeListStr = ""
	}
	includeNamespaces := strings.Split(namespaceLists.IncludeListStr, ",")
	excludeNamespaces := strings.Split(namespaceLists.ExcludeListStr, ",")
	namespaceList, err := kubeClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to retrieve namespaces: %v\n", err)
		os.Exit(1)
	}
	if namespaceLists.IncludeListStr != "" {
		for _, ns := range namespaceList.Items {
			namespacesMap[ns.Name] = false
		}
		for _, ns := range includeNamespaces {
			if _, exists := namespacesMap[ns]; exists {
				namespacesMap[ns] = true
			} else {
				fmt.Fprintf(os.Stderr, "namespace [%s] not found\n", ns)
			}
		}
	} else {
		for _, ns := range namespaceList.Items {
			namespacesMap[ns.Name] = true
		}
		for _, ns := range excludeNamespaces {
			if _, exists := namespacesMap[ns]; exists {
				namespacesMap[ns] = false
			}
		}
	}
	for ns := range namespacesMap {
		if namespacesMap[ns] {
			namespaces = append(namespaces, ns)
		}
	}
	return namespaces
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

func FormatOutputAll(namespace string, allDiffs []ResourceDiff) string {
	i := 0
	var buf bytes.Buffer
	table := tablewriter.NewWriter(&buf)
	table.SetHeader([]string{"#", "Resource Type", "Resource Name"})

	// TODO parse resourceType, diff

	allEmpty := true
	for _, data := range allDiffs {
		if len(data.diff) == 0 {
			continue
		}

		allEmpty = false
		for _, val := range data.diff {
			row := []string{fmt.Sprintf("%d", i+1), data.resourceType, val}
			table.Append(row)
			i += 1
		}
	}

	if allEmpty {
		return fmt.Sprintf("No unused resources found in the namespace: %s", namespace)
	}

	table.Render()
	return fmt.Sprintf("Unused Resources in Namespace: %s\n%s", namespace, buf.String())
}

// TODO create formatter by resource "#", "Resource Name", "Namespace"

func CalculateResourceDifference(usedResourceNames []string, allResourceNames []string) []string {
	var difference []string
	for _, name := range allResourceNames {
		found := false
		for _, usedName := range usedResourceNames {
			if name == usedName {
				found = true
				break
			}
		}
		if !found {
			difference = append(difference, name)
		}
	}
	return difference
}
