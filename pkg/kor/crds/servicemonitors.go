package crds

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func GetUnusedServiceMontors(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts common.Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)
	diff, err := processClusterRoles(clientset, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to process cluster role : %v\n", err)
	}
	if opts.DeleteFlag {
		if diff, err = DeleteResource(diff, clientset, "", "ClusterRole", opts.NoInteractive); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete clusterRole %s : %v\n", diff, err)
		}
	}
	switch opts.GroupBy {
	case "namespace":
		resources[""] = make(map[string][]ResourceInfo)
		resources[""]["ClusterRole"] = diff
	case "resource":
		appendResources(resources, "ClusterRole", "", diff)
	}

	var outputBuffer bytes.Buffer
	var jsonResponse []byte
	switch outputFormat {
	case "table":
		outputBuffer = FormatOutput(resources, opts)
	case "json", "yaml":
		var err error
		if jsonResponse, err = json.MarshalIndent(resources, "", "  "); err != nil {
			return "", err
		}
	}

	unusedClusterRoles, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedClusterRoles, nil
}
