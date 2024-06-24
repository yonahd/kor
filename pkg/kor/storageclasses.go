package kor

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	"github.com/yonahd/kor/pkg/filters"
)

//go:embed exceptions/storageclasses/storageclasses.json
var storageClassesConfig []byte

func retrieveUsedStorageClasses(clientset kubernetes.Interface) ([]string, error) {
	pvs, err := clientset.CoreV1().PersistentVolumes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Failed to list PVs: %v\n", err)
		os.Exit(1)
	}

	pvcs, err := clientset.CoreV1().PersistentVolumeClaims("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Failed to list PVCs: %v\n", err)
		os.Exit(1)
	}

	var usedStorageClasses []string

	// Iterate through each PV and check for StorageClass usage
	for _, pv := range pvs.Items {
		if pv.Spec.StorageClassName != "" {
			usedStorageClasses = append(usedStorageClasses, pv.Spec.StorageClassName)
		}
	}

	// Iterate through each PVC and check for StorageClass usage
	for _, pvc := range pvcs.Items {
		if pvc.Spec.StorageClassName != nil {
			usedStorageClasses = append(usedStorageClasses, *pvc.Spec.StorageClassName)
		}
	}

	return usedStorageClasses, err
}

func processStorageClasses(clientset kubernetes.Interface, filterOpts *filters.Options) ([]ResourceInfo, error) {
	scs, err := clientset.StorageV1().StorageClasses().List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	config, err := unmarshalConfig(storageClassesConfig)
	if err != nil {
		return nil, err
	}

	var unusedStorageClasses []ResourceInfo
	storageClassNames := make([]string, 0, len(scs.Items))

	for _, sc := range scs.Items {
		if pass := filters.KorLabelFilter(&sc, &filters.Options{}); pass {
			continue
		}

		if sc.Labels["kor/used"] == "false" {
			unusedStorageClasses = append(unusedStorageClasses, ResourceInfo{Name: sc.Name, Reason: "Marked with unused label"})
			continue
		}

		exceptionFound, err := isResourceException(sc.Name, sc.Namespace, config.ExceptionStorageClasses)
		if err != nil {
			return nil, err
		}

		if exceptionFound {
			continue
		}

		storageClassNames = append(storageClassNames, sc.Name)
	}

	usedStorageClasses, err := retrieveUsedStorageClasses(clientset)
	if err != nil {
		return nil, err
	}

	diff := CalculateResourceDifference(usedStorageClasses, storageClassNames)
	for _, name := range diff {
		unusedStorageClasses = append(unusedStorageClasses, ResourceInfo{Name: name, Reason: "Not in Use"})
	}
	return unusedStorageClasses, nil
}

func GetUnusedStorageClasses(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)
	diff, err := processStorageClasses(clientset, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to process storageClasses: %v\n", err)
	}
	if opts.DeleteFlag {
		if diff, err = DeleteResource(diff, clientset, "", "StorageClass", opts.NoInteractive); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete StorageClass %s: %v\n", diff, err)
		}
	}
	switch opts.GroupBy {
	case "namespace":
		resources[""] = make(map[string][]ResourceInfo)
		resources[""]["StorageClass"] = diff
	case "resource":
		appendResources(resources, "StorageClass", "", diff)
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

	unusedStorageClasses, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedStorageClasses, nil
}
