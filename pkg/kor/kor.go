package kor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/olekukonko/tablewriter"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/yaml"
)

type ExceptionResource struct {
	Namespace    string
	ResourceName string
}
type IncludeExcludeLists struct {
	IncludeListStr string
	ExcludeListStr string
}

type Config struct {
	ExceptionClusterRoles    []ExceptionResource `json:"exceptionClusterRoles"`
	ExceptionConfigMaps      []ExceptionResource `json:"exceptionConfigMaps"`
	ExceptionCrds            []ExceptionResource `json:"exceptionCrds"`
	ExceptionDaemonSets      []ExceptionResource `json:"exceptionDaemonSets"`
	ExceptionRoles           []ExceptionResource `json:"exceptionRoles"`
	ExceptionSecrets         []ExceptionResource `json:"exceptionSecrets"`
	ExceptionServiceAccounts []ExceptionResource `json:"exceptionServiceAccounts"`
	ExceptionServices        []ExceptionResource `json:"exceptionServices"`
	ExceptionStorageClasses  []ExceptionResource `json:"exceptionStorageClasses"`
	// Add other configurations if needed
}

type Opts struct {
	DeleteFlag    bool
	NoInteractive bool
	Verbose       bool
	WebhookURL    string
	Channel       string
	Token         string
	GroupBy       string
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

func appendResources(resources map[string]map[string][]string, resourceType, namespace string, diff []string) {
	for _, d := range diff {
		if _, ok := resources[resourceType]; !ok {
			resources[resourceType] = make(map[string][]string)
		}
		resources[resourceType][namespace] = append(resources[resourceType][namespace], d)
	}
}

func getTableHeader(groupBy string) []string {
	switch groupBy {
	case "namespace":
		return []string{
			"#",
			"RESOURCE TYPE",
			"RESOURCE NAME",
		}
	case "resource":
		return []string{
			"#",
			"NAMESPACE",
			"RESOURCE NAME",
		}
	default:
		return nil
	}
}

func getTableRow(index int, columns ...string) []string {
	row := make([]string, 0, len(columns)+1)
	row = append(row, fmt.Sprintf("%d", index+1))
	row = append(row, columns...)
	return row
}

// FormatOutput formats the output based on the group by option
func FormatOutput(resources map[string]map[string][]string, opts Opts) bytes.Buffer {
	var output bytes.Buffer
	switch opts.GroupBy {
	case "namespace":
		for namespace, diffs := range resources {
			output.WriteString(formatOutputForNamespace(namespace, diffs, opts))
		}
	case "resource":
		for resource, diffs := range resources {
			output.WriteString(formatOutputForResource(resource, diffs, opts))
		}
	}
	return output
}

func formatOutputForResource(resource string, resources map[string][]string, opts Opts) string {
	if len(resources) == 0 {
		if opts.Verbose {
			return fmt.Sprintf("No unused %ss found\n", resource)
		}
		return ""
	}
	var buf bytes.Buffer
	table := tablewriter.NewWriter(&buf)
	table.SetHeader(getTableHeader(opts.GroupBy))
	var index int
	for ns, diffs := range resources {
		for _, d := range diffs {
			row := getTableRow(index, ns, d)
			table.Append(row)
			index++
		}
	}
	table.Render()
	return fmt.Sprintf("Unused %ss:\n%s\n", resource, buf.String())
}

func formatOutputForNamespace(namespace string, resources map[string][]string, opts Opts) string {
	var buf bytes.Buffer
	table := tablewriter.NewWriter(&buf)
	table.SetHeader(getTableHeader(opts.GroupBy))
	allEmpty := true
	var index int
	for resourceType, diff := range resources {
		for _, val := range diff {
			row := getTableRow(index, resourceType, val)
			table.Append(row)
			allEmpty = false
			index++
		}
	}
	if allEmpty {
		if opts.Verbose {
			return fmt.Sprintf("No unused resources found in the namespace: %q\n", namespace)
		}
		return ""
	}
	table.Render()
	return fmt.Sprintf("Unused resources in namespace: %q\n%s\n", namespace, buf.String())
}

func FormatOutputAll(namespace string, allDiffs []ResourceDiff, opts Opts) string {
	var buf bytes.Buffer
	table := tablewriter.NewWriter(&buf)
	table.SetHeader(getTableHeader(opts.GroupBy))
	allEmpty := true
	var index int
	for _, data := range allDiffs {
		for _, val := range data.diff {
			row := getTableRow(index, data.resourceType, val)
			table.Append(row)
			allEmpty = false
			index++
		}
	}
	if allEmpty {
		if opts.Verbose {
			return fmt.Sprintf("No unused resources found in the namespace: %q\n", namespace)
		}
		return ""
	}
	table.Render()
	return fmt.Sprintf("Unused resources in namespace: %q\n%s\n", namespace, buf.String())
}

// TODO create formatter by resource "#", "Resource Name", "Namespace"
// TODO Functions that use this object are accompanied by repeated data acquisition operations and can be optimized.
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
	switch outputFormat {
	case "table":
		if opts.WebhookURL == "" || opts.Channel == "" || opts.Token != "" {
			return outputBuffer.String(), nil
		}
		if err := SendToSlack(SlackMessage{}, opts, outputBuffer.String()); err != nil {
			return "", fmt.Errorf("failed to send message to slack: %w", err)
		}
	case "yaml":
		yamlResponse, err := yaml.JSONToYAML(jsonResponse)
		if err != nil {
			return "", fmt.Errorf("failed to convert json to yaml: %w", err)
		}
		return string(yamlResponse), nil
	}
	return string(jsonResponse), nil
}

func isResourceException(resourceName, namespace string, exceptions []ExceptionResource) bool {
	var match bool
	for _, e := range exceptions {
		if e.ResourceName == resourceName && e.Namespace == namespace {
			match = true
			break
		}
		if strings.HasSuffix(e.ResourceName, "*") {
			resourceNameMatched := strings.HasPrefix(resourceName, strings.TrimSuffix(e.ResourceName, "*"))
			if resourceNameMatched && e.Namespace == namespace {
				match = true
				break
			}
		}
	}
	return match
}

func unmarshalConfig(data []byte) (*Config, error) {
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
