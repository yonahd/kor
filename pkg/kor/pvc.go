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

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
	"github.com/yonahd/kor/pkg/kor/externaldeps"
)

func retrieveUsedPvcs(clientset kubernetes.Interface, namespace string) ([]string, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("Failed to list Pods: %v\n", err)
		os.Exit(1)
	}
	var usedPvcs []string
	// Iterate through each Pod and check for PVC usage
	for _, pod := range pods.Items {
		for _, volume := range pod.Spec.Volumes {
			if volume.PersistentVolumeClaim != nil {
				usedPvcs = append(usedPvcs, volume.PersistentVolumeClaim.ClaimName)
			}
			// Include ephemeral PVC
			if volume.Ephemeral != nil && volume.Ephemeral.VolumeClaimTemplate != nil {
				// https://kubernetes.io/docs/concepts/storage/ephemeral-volumes/#persistentvolumeclaim-naming
				usedPvcs = append(usedPvcs, pod.GetObjectMeta().GetName()+"-"+volume.Name)
			}
		}
	}
	return usedPvcs, err
}

func retrieveUsedPvcsFromExternalCRDs(clientset kubernetes.Interface, namespace string) ([]string, error) {
	registry := externaldeps.GetGlobalRegistry()
	dynamicClient := GetDynamicClient("")

	refs, err := registry.ScanNamespace(context.TODO(), namespace, clientset, dynamicClient)
	if err != nil {
		return nil, err
	}

	return RemoveDuplicatesAndSort(refs.PVCs), nil
}

func processNamespacePvcs(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options, opts common.Opts) ([]ResourceInfo, error) {
	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	var unusedPvcNames []string
	pvcNames := make([]string, 0, len(pvcs.Items))
	for _, pvc := range pvcs.Items {
		// Skip resources with ownerReferences if the general flag is set
		if filterOpts.IgnoreOwnerReferences && len(pvc.OwnerReferences) > 0 {
			continue
		}

		if pass, _ := filter.SetObject(&pvc).Run(filterOpts); pass {
			continue
		}

		if pvc.Labels["kor/used"] == "false" {
			unusedPvcNames = append(unusedPvcNames, pvc.Name)
			continue
		}

		pvcNames = append(pvcNames, pvc.Name)
	}

	usedPvcs, err := retrieveUsedPvcs(clientset, namespace)
	if err != nil {
		return nil, err
	}

	// Retrieve PVCs referenced by external CRDs (like Argo WorkflowTemplates)
	externalPvcs, err := retrieveUsedPvcsFromExternalCRDs(clientset, namespace)
	if err != nil {
		return nil, err
	}

	// Combine all used PVCs
	allUsedPvcs := append(usedPvcs, externalPvcs...)
	allUsedPvcs = RemoveDuplicatesAndSort(allUsedPvcs)

	var diff []ResourceInfo
	for _, name := range CalculateResourceDifference(allUsedPvcs, pvcNames) {
		reason := "PVC is not in use"
		diff = append(diff, ResourceInfo{Name: name, Reason: reason})
	}

	for _, name := range unusedPvcNames {
		reason := "Marked with unused label"
		diff = append(diff, ResourceInfo{Name: name, Reason: reason})
	}

	if opts.DeleteFlag {
		if diff, err = DeleteResource(diff, clientset, namespace, "PVC", opts.NoInteractive); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete PVC %s in namespace %s: %v\n", diff, namespace, err)
		}
	}
	return diff, nil
}

func GetUnusedPvcs(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts common.Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)
	for _, namespace := range filterOpts.Namespaces(clientset) {
		diff, err := processNamespacePvcs(clientset, namespace, filterOpts, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		switch opts.GroupBy {
		case "namespace":
			resources[namespace] = make(map[string][]ResourceInfo)
			resources[namespace]["Pvc"] = diff
		case "resource":
			appendResources(resources, "Pvc", namespace, diff)
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

	unusedPvcs, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedPvcs, nil
}
