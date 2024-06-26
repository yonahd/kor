package kor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	"github.com/yonahd/kor/pkg/filters"
)

func validateServiceBackend(clientset kubernetes.Interface, namespace string, backend *v1.IngressBackend) bool {
	if backend.Service != nil {
		serviceName := backend.Service.Name

		_, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), serviceName, metav1.GetOptions{})
		if err != nil {
			return false
		}
	}
	return true
}

func retrieveUsedIngress(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ([]string, error) {
	ingresses, err := clientset.NetworkingV1().Ingresses(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	usedIngresses := []string{}

	for _, ingress := range ingresses.Items {
		if pass, _ := filter.SetObject(&ingress).Run(filterOpts); pass {
			continue
		}

		used := true

		if ingress.Spec.DefaultBackend != nil {
			used = validateServiceBackend(clientset, namespace, ingress.Spec.DefaultBackend)
		}
		for _, rule := range ingress.Spec.Rules {
			if rule.HTTP == nil {
				used = true
				break
			}
			for _, path := range rule.HTTP.Paths {
				used = validateServiceBackend(clientset, namespace, &path.Backend)
				if used {
					break
				}
			}
			if used {
				break
			}
		}
		if used {
			usedIngresses = append(usedIngresses, ingress.Name)
		}
	}
	return usedIngresses, nil
}

func retrieveIngressNames(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ([]string, []string, error) {
	ingresses, err := clientset.NetworkingV1().Ingresses(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, nil, err
	}

	var unusedIngressNames []string
	names := make([]string, 0, len(ingresses.Items))

	for _, ingress := range ingresses.Items {
		if pass, _ := filter.SetObject(&ingress).Run(filterOpts); pass {
			continue
		}

		if ingress.Labels["kor/used"] == "false" {
			unusedIngressNames = append(unusedIngressNames, ingress.Name)
			continue
		}
		names = append(names, ingress.Name)
	}
	return names, unusedIngressNames, nil
}

func processNamespaceIngresses(clientset kubernetes.Interface, namespace string, filterOpts *filters.Options) ([]ResourceInfo, error) {
	usedIngresses, err := retrieveUsedIngress(clientset, namespace, filterOpts)
	if err != nil {
		return nil, err
	}
	ingressNames, unusedIngressNames, err := retrieveIngressNames(clientset, namespace, filterOpts)
	if err != nil {
		return nil, err
	}

	var diff []ResourceInfo

	for _, name := range CalculateResourceDifference(usedIngresses, ingressNames) {
		reason := "Ingress does not have a valid backend service"
		diff = append(diff, ResourceInfo{Name: name, Reason: reason})
	}

	for _, name := range unusedIngressNames {
		reason := "Marked with unused label"
		diff = append(diff, ResourceInfo{Name: name, Reason: reason})
	}

	return diff, nil

}

func GetUnusedIngresses(filterOpts *filters.Options, clientset kubernetes.Interface, outputFormat string, opts Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)
	for _, namespace := range filterOpts.Namespaces(clientset) {
		diff, err := processNamespaceIngresses(clientset, namespace, filterOpts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		if opts.DeleteFlag {
			if diff, err = DeleteResource(diff, clientset, namespace, "Ingress", opts.NoInteractive); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to delete Ingress %s in namespace %s: %v\n", diff, namespace, err)
			}
		}
		switch opts.GroupBy {
		case "namespace":
			resources[namespace] = make(map[string][]ResourceInfo)
			resources[namespace]["Ingress"] = diff
		case "resource":
			appendResources(resources, "Ingress", namespace, diff)
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

	unusedIngresses, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedIngresses, nil
}
