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
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/yaml"
)

type ExceptionResource struct {
	ResourceName string
	Namespace    string
}
type IncludeExcludeLists struct {
	IncludeListStr string
	ExcludeListStr string
}

type Opts struct {
	DeleteFlag    bool
	NoInteractive bool
	Verbose       bool
	WebhookURL    string
	Channel       string
	Token         string
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

func GetConfig(kubeconfig string) (*rest.Config, error) {
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token"); err == nil {
		return rest.InClusterConfig()
	}

	if kubeconfig == "" {
		if configEnv := os.Getenv("KUBECONFIG"); configEnv != "" {
			kubeconfig = configEnv
		} else {
			kubeconfig = GetKubeConfigPath()
		}
	}

	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}

func GetKubeClient(kubeconfig string) *kubernetes.Clientset {
	config, err := GetConfig(kubeconfig)
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

func GetAPIExtensionsClient(kubeconfig string) *apiextensionsclientset.Clientset {
	config, err := GetConfig(kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load kubeconfig: %v\n", err)
		os.Exit(1)
	}

	clientset, err := apiextensionsclientset.NewForConfig(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create Kubernetes client: %v\n", err)
		os.Exit(1)
	}
	return clientset
}

func GetDynamicClient(kubeconfig string) *dynamic.DynamicClient {
	config, err := GetConfig(kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load kubeconfig: %v\n", err)
		os.Exit(1)
	}

	clientset, err := dynamic.NewForConfig(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create Kubernetes client: %v\n", err)
		os.Exit(1)
	}
	return clientset
}

func SetNamespaceList(namespaceLists IncludeExcludeLists, clientset kubernetes.Interface) []string {
	namespaces := make([]string, 0)
	namespacesMap := make(map[string]bool)
	if namespaceLists.IncludeListStr != "" && namespaceLists.ExcludeListStr != "" {
		fmt.Fprintf(os.Stderr, "Exclude namespaces can't be used together with include namespaces. Ignoring --exclude-namespace(-e) flag\n")
		namespaceLists.ExcludeListStr = ""
	}
	includeNamespaces := strings.Split(namespaceLists.IncludeListStr, ",")
	excludeNamespaces := strings.Split(namespaceLists.ExcludeListStr, ",")
	namespaceList, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
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

func FormatOutput(namespace string, resources []string, resourceType string, opts Opts) string {
	if opts.Verbose && len(resources) == 0 {
		return fmt.Sprintf("No unused %s found in the namespace: %s \n", resourceType, namespace)
	} else if len(resources) == 0 {
		return ""
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

func FormatOutputFromMap(namespace string, allDiffs map[string][]string, opts Opts) string {
	i := 0
	var buf bytes.Buffer
	table := tablewriter.NewWriter(&buf)
	table.SetHeader([]string{"#", "Resource Type", "Resource Name"})

	// TODO parse resourceType, diff

	allEmpty := true
	for resourceType, diff := range allDiffs {
		if len(diff) == 0 {
			continue
		}

		allEmpty = false
		for _, val := range diff {
			row := []string{fmt.Sprintf("%d", i+1), resourceType, val}
			table.Append(row)
			i += 1
		}
	}

	if opts.Verbose && allEmpty {
		return fmt.Sprintf("No unused resources found in the namespace: %s", namespace)
	} else if allEmpty {
		return ""
	}

	table.Render()
	if namespace == "" {
		return fmt.Sprintf("Unused CRDs: \n%s", buf.String())
	}
	return fmt.Sprintf("Unused Resources in Namespace: %s\n%s", namespace, buf.String())
}

func FormatOutputAll(namespace string, allDiffs []ResourceDiff, opts Opts) string {
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

	if opts.Verbose && allEmpty {
		return fmt.Sprintf("No unused resources found in the namespace: %s", namespace)
	} else if allEmpty {
		return ""
	}

	table.Render()
	if namespace == "" {
		return fmt.Sprintf("Unused CRDs: \n%s", buf.String())
	}
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

func unusedResourceFormatter(outputFormat string, outputBuffer bytes.Buffer, opts Opts, jsonResponse []byte) (string, error) {
	if outputFormat == "table" {

		if opts.WebhookURL != "" || opts.Channel != "" && opts.Token != "" {
			if err := SendToSlack(SlackMessage{}, opts, outputBuffer.String()); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to send message to slack: %v\n", err)
				os.Exit(1)
			}
		} else {
			return outputBuffer.String(), nil
		}
	} else {
		if outputFormat == "yaml" {
			yamlResponse, err := yaml.JSONToYAML(jsonResponse)
			if err != nil {
				fmt.Printf("err: %v\n", err)
			}
			return string(yamlResponse), nil
		}
	}
	return string(jsonResponse), nil
}
