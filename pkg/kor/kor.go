package kor

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"

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
	PrintReason   bool
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

func ConvertToResourceInfo(resources []ResourceInfo) []string {
	resourceInfos := make([]string, len(resources))
	for i := range resources {
		resourceInfos[i] = resources[i].Name
	}
	return resourceInfos
}

func FormatOutput(namespace string, resources interface{}, resourceType string, opts Opts) string {
	var resourceNames []string
	switch res := resources.(type) {
	case []string:
		resourceNames = res
	default:
		return "Invalid resource type"
	}

	if opts.Verbose && len(resourceNames) == 0 {
		return fmt.Sprintf("No unused %s found in the namespace: %s \n", resourceType, namespace)
	} else if len(resourceNames) == 0 {
		return ""
	}

	var buf bytes.Buffer
	table := tablewriter.NewWriter(&buf)
	table.SetHeader([]string{"#", "Resource Name"})

	for i, name := range resourceNames {
		table.Append([]string{fmt.Sprintf("%d", i+1), name})
	}

	table.Render()
	return fmt.Sprintf("Unused %s in Namespace: %s\n%s", resourceType, namespace, buf.String())
}

func FormatEnrichedOutput(namespace string, resources interface{}, resourceType string, opts Opts) string {
	var enrichedResources []ResourceInfo
	switch res := resources.(type) {
	case []ResourceInfo:
		enrichedResources = res
	default:
		return "Invalid resource type"
	}

	if opts.Verbose && len(enrichedResources) == 0 {
		return fmt.Sprintf("No unused %s found in the namespace: %s \n", resourceType, namespace)
	} else if len(enrichedResources) == 0 {
		return ""
	}

	var buf bytes.Buffer
	table := tablewriter.NewWriter(&buf)
	table.SetHeader([]string{"#", "Resource Name", "Reason"})

	for i, info := range enrichedResources {
		table.Append([]string{fmt.Sprintf("%d", i+1), info.Name, info.Reason})
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
		return fmt.Sprintf("Unused %ss: \n%s", allDiffs[0].resourceType, buf.String())
	}
	return fmt.Sprintf("Unused Resources in Namespace: %s\n%s", namespace, buf.String())
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
