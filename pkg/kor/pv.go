package kor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	"github.com/yonahd/kor/pkg/filters"
)

func processPvs(clientset kubernetes.Interface, filterOpts *filters.Options) ([]string, error) {
	pvs, err := clientset.CoreV1().PersistentVolumes().List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	var unusedPvs []string

	for _, pv := range pvs.Items {
		if pass := filters.KorLabelFilter(&pv, &filters.Options{}); pass {
			continue
		}

		if pv.Labels["kor/used"] == "false" {
			unusedPvs = append(unusedPvs, pv.Name)
			continue
		}

		if pv.Status.Phase != "Bound" {
			unusedPvs = append(unusedPvs, pv.Name)
		}

	}

	return unusedPvs, nil

}

func GetUnusedPvs(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts Opts) (string, error) {
	resources := make(map[string]map[string][]string)
	diff, err := processPvs(clientset, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to process pvs: %v\n", err)
	}
	switch opts.GroupBy {
	case "namespace":
		resources[""] = make(map[string][]string)
		resources[""]["Pv"] = diff
	case "resource":
		appendResources(resources, "Pv", "", diff)
	}
	if opts.DeleteFlag {
		if diff, err = DeleteResource(diff, clientset, "", "PV", opts.NoInteractive); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete PV %s: %v\n", diff, err)
		}
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

	unusedPvs, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedPvs, nil
}
