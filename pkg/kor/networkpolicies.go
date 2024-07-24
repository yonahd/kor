package kor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"

	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

const (
	unusedLabelReason         = "Marked with unused label"
	noPodAppliedReason        = "NetworkPolicy applies to 0 pods"
	noPodAppliedByRulesReason = "NetworkPolicy Ingress and Egress rules apply to 0 pods"
)

func retrievePodsForSelector(clientset kubernetes.Interface, namespace string, selector *metav1.LabelSelector) ([]v1.Pod, error) {
	labelSelector, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return nil, err
	}
	podList, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labelSelector.String(),
	})
	if err != nil {
		return nil, err
	}

	return podList.Items, nil
}

func isAnyPodMatchedInSources(clientset kubernetes.Interface, sources []networkingv1.NetworkPolicyPeer) (bool, error) {
	// If this field is empty or missing, this rule matches all pods
	if len(sources) == 0 {
		return true, nil
	}

	for _, netpolPeer := range sources {
		// If ipBlock is specified, assume the source is in use
		if netpolPeer.IPBlock != nil {
			return true, nil
		}

		labelSelector, err := metav1.LabelSelectorAsSelector(netpolPeer.NamespaceSelector)
		if err != nil {
			return false, err
		}

		nsList, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{
			LabelSelector: labelSelector.String(),
		})
		if err != nil {
			return false, err
		}

		for _, ns := range nsList.Items {
			podList, err := retrievePodsForSelector(clientset, ns.Name, netpolPeer.PodSelector)
			if err != nil {
				return false, err
			}

			if len(podList) > 0 {
				return true, nil
			}
		}
	}

	return false, nil
}

func isAnyIngressRuleUsed(clientset kubernetes.Interface, netpol networkingv1.NetworkPolicy) (bool, error) {
	// Deny all ingress traffic
	if len(netpol.Spec.Ingress) == 0 && slices.Contains(netpol.Spec.PolicyTypes, networkingv1.PolicyTypeIngress) {
		return true, nil
	}
	for _, ingressRule := range netpol.Spec.Ingress {
		podsMatched, err := isAnyPodMatchedInSources(clientset, ingressRule.From)
		if err != nil {
			return false, err
		}

		if podsMatched {
			return true, nil
		}
	}

	return false, nil
}

func isAnyEgressRuleUsed(clientset kubernetes.Interface, netpol networkingv1.NetworkPolicy) (bool, error) {
	// Deny all egress traffic
	if len(netpol.Spec.Egress) == 0 && slices.Contains(netpol.Spec.PolicyTypes, networkingv1.PolicyTypeEgress) {
		return true, nil
	}

	for _, egressRule := range netpol.Spec.Egress {
		podsMatched, err := isAnyPodMatchedInSources(clientset, egressRule.To)
		if err != nil {
			return false, err
		}

		if podsMatched {
			return true, nil
		}
	}

	return false, nil
}

func processNamespaceNetworkPolicies(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ([]ResourceInfo, error) {
	netpolList, err := clientset.NetworkingV1().NetworkPolicies(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	var unusedNetpols []ResourceInfo

	for _, netpol := range netpolList.Items {
		if pass, _ := filter.SetObject(&netpol).Run(filterOpts); pass {
			continue
		}

		if netpol.Labels["kor/used"] == "false" {
			unusedNetpols = append(unusedNetpols, ResourceInfo{Name: netpol.Name, Reason: unusedLabelReason})
			continue
		}

		pods, err := retrievePodsForSelector(clientset, namespace, &netpol.Spec.PodSelector)
		if err != nil {
			return nil, err
		}

		if len(pods) == 0 {
			unusedNetpols = append(unusedNetpols, ResourceInfo{Name: netpol.Name, Reason: noPodAppliedReason})
			continue
		}

		if used, err := isAnyIngressRuleUsed(clientset, netpol); err != nil {
			return nil, err
		} else if used {
			continue
		}

		if used, err := isAnyEgressRuleUsed(clientset, netpol); err != nil {
			return nil, err
		} else if used {
			continue
		}

		unusedNetpols = append(unusedNetpols, ResourceInfo{Name: netpol.Name, Reason: noPodAppliedByRulesReason})
	}

	return unusedNetpols, nil
}

func GetUnusedNetworkPolicies(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts common.Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)

	for _, namespace := range filterOpts.Namespaces(clientset) {
		diff, err := processNamespaceNetworkPolicies(clientset, namespace, filterOpts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		if opts.DeleteFlag {
			if diff, err := DeleteResource(diff, clientset, namespace, "NetworkPolicy", opts.NoInteractive); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to delete NetworkPolicy %s in namespace %s: %v\n", diff, namespace, err)
			}
		}
		switch opts.GroupBy {
		case "namespace":
			resources[namespace] = make(map[string][]ResourceInfo)
			resources[namespace]["NetworkPolicy"] = diff
		case "resource":
			appendResources(resources, "NetworkPolicy", namespace, diff)
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

	unusedNetworkPolicies, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedNetworkPolicies, nil
}
