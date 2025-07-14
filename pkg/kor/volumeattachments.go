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
)

func processVolumeAttachments(clientset kubernetes.Interface, filterOpts *filters.Options) ([]ResourceInfo, error) {
	vaList, err := clientset.StorageV1().VolumeAttachments().List(context.TODO(), metav1.ListOptions{
		LabelSelector: filterOpts.IncludeLabels,
	})
	if err != nil {
		return nil, err
	}

	var unusedVAtts []ResourceInfo

	for _, va := range vaList.Items {
		// Skip resources with ownerReferences if the general flag is set
		if filterOpts.IgnoreOwnerReferences && len(va.OwnerReferences) > 0 {
			continue
		}

		if pass, _ := filter.SetObject(&va).Run(filterOpts); pass {
			continue
		}

		if va.Labels["kor/used"] == "false" {
			reason := "Marked with unused label"
			unusedVAtts = append(unusedVAtts, ResourceInfo{Name: va.Name, Reason: reason})
			continue
		}

		pvName := va.Spec.Source.PersistentVolumeName
		if pvName == nil || *pvName == "" {
			reason := "No PersistentVolume specified in VolumeAttachment"
			unusedVAtts = append(unusedVAtts, ResourceInfo{Name: va.Name, Reason: reason})
			continue
		}
		if _, err := clientset.CoreV1().PersistentVolumes().Get(context.TODO(), *pvName, metav1.GetOptions{}); err != nil {
			reason := fmt.Sprintf("PersistentVolume %s does not exist", *pvName)
			unusedVAtts = append(unusedVAtts, ResourceInfo{Name: va.Name, Reason: reason})
			continue
		}

		nodeName := va.Spec.NodeName
		if nodeName == "" {
			reason := "No node specified in VolumeAttachment"
			unusedVAtts = append(unusedVAtts, ResourceInfo{Name: va.Name, Reason: reason})
			continue
		}
		if _, err := clientset.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{}); err != nil {
			reason := fmt.Sprintf("Node %s does not exist", nodeName)
			unusedVAtts = append(unusedVAtts, ResourceInfo{Name: va.Name, Reason: reason})
			continue
		}

		attacher := va.Spec.Attacher
		if attacher == "" {
			reason := "No attacher specified in VolumeAttachment"
			unusedVAtts = append(unusedVAtts, ResourceInfo{Name: va.Name, Reason: reason})
			continue
		}
		if _, err := clientset.StorageV1().CSIDrivers().Get(context.TODO(), attacher, metav1.GetOptions{}); err != nil {
			reason := fmt.Sprintf("CSIDriver %s does not exist", attacher)
			unusedVAtts = append(unusedVAtts, ResourceInfo{Name: va.Name, Reason: reason})
			continue
		}

	}

	return unusedVAtts, nil
}

func GetUnusedVolumeAttachments(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts common.Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)
	diff, err := processVolumeAttachments(clientset, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to process volume attachments: %v\n", err)
	}
	if opts.DeleteFlag {
		if diff, err = DeleteResource(diff, clientset, "", "VolumeAttachment", opts.NoInteractive); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete VolumeAttachments: %v\n", err)
		}
	}
	switch opts.GroupBy {
	case "namespace":
		resources[""] = make(map[string][]ResourceInfo)
		resources[""]["VolumeAttachment"] = diff
	case "resource":
		appendResources(resources, "VolumeAttachment", "", diff)
	}

	var outputBuffer bytes.Buffer
	var jsonResponse []byte
	switch outputFormat {
	case "table":
		outputBuffer = FormatOutput(resources, opts)
	case "json", "yaml":
		if jsonResponse, err = json.MarshalIndent(resources, "", "  "); err != nil {
			return "", err
		}
	}

	unusedVAtts, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedVAtts, nil
}
