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
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayclientset "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

//go:embed exceptions/gateways/gateways.json
var gatewaysConfig []byte

func processNamespaceGateways(clientset kubernetes.Interface, gatewayClient gatewayclientset.Interface, namespace string, filterOpts *filters.Options, opts common.Opts) ([]ResourceInfo, error) {
	gateways, err := gatewayClient.GatewayV1().Gateways(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	config, err := unmarshalConfig(gatewaysConfig)
	if err != nil {
		return nil, err
	}

	var unusedGateways []ResourceInfo

	for _, gateway := range gateways.Items {
		// Skip resources with ownerReferences if the general flag is set
		if filterOpts.IgnoreOwnerReferences && len(gateway.OwnerReferences) > 0 {
			continue
		}

		if pass, _ := filter.SetObject(&gateway).Run(filterOpts); pass {
			continue
		}

		if gateway.Labels["kor/used"] == "false" {
			reason := "Marked with unused label"
			unusedGateways = append(unusedGateways, ResourceInfo{Name: gateway.Name, Reason: reason})
			continue
		}

		exceptionFound, err := isResourceException(gateway.Name, gateway.Namespace, config.ExceptionGateways)
		if err != nil {
			return nil, err
		}

		if exceptionFound {
			continue
		}

		// Check if the GatewayClass exists
		gatewayClassExists, err := checkGatewayClassExists(gatewayClient, gateway.Spec.GatewayClassName)
		if err != nil {
			return nil, err
		}

		if !gatewayClassExists {
			reason := "Gateway references a non-existing GatewayClass"
			unusedGateways = append(unusedGateways, ResourceInfo{Name: gateway.Name, Reason: reason})
			continue
		}

		// Check if the Gateway has at least one attached route
		hasRoutes, err := checkGatewayHasRoutes(gatewayClient, &gateway)
		if err != nil {
			return nil, err
		}

		if !hasRoutes {
			reason := "Gateway has no attached routes"
			unusedGateways = append(unusedGateways, ResourceInfo{Name: gateway.Name, Reason: reason})
		}
	}

	if opts.DeleteFlag {
		if unusedGateways, err = DeleteResource(unusedGateways, clientset, namespace, "Gateway", opts.NoInteractive); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete Gateway %s in namespace %s: %v\n", unusedGateways, namespace, err)
		}
	}

	return unusedGateways, nil
}

func checkGatewayClassExists(gatewayClient gatewayclientset.Interface, gatewayClassName gatewayv1.ObjectName) (bool, error) {
	_, err := gatewayClient.GatewayV1().GatewayClasses().Get(context.TODO(), string(gatewayClassName), metav1.GetOptions{})
	if err != nil {
		return false, nil // GatewayClass doesn't exist
	}
	return true, nil
}

func checkGatewayHasRoutes(gatewayClient gatewayclientset.Interface, gateway *gatewayv1.Gateway) (bool, error) {
	// Check for HTTPRoutes
	httpRoutes, err := gatewayClient.GatewayV1().HTTPRoutes(gateway.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return false, err
	}

	for _, route := range httpRoutes.Items {
		for _, parentRef := range route.Spec.ParentRefs {
			if parentRef.Name == gatewayv1.ObjectName(gateway.Name) {
				// Check if the namespace matches (default to same namespace if not specified)
				routeNamespace := gateway.Namespace
				if parentRef.Namespace != nil {
					routeNamespace = string(*parentRef.Namespace)
				}
				if routeNamespace == gateway.Namespace {
					return true, nil
				}
			}
		}
	}

	// Check for TCPRoutes
	tcpRoutes, err := gatewayClient.GatewayV1alpha2().TCPRoutes(gateway.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return false, err
	}

	for _, route := range tcpRoutes.Items {
		for _, parentRef := range route.Spec.ParentRefs {
			if parentRef.Name == gatewayv1.ObjectName(gateway.Name) {
				// Check if the namespace matches (default to same namespace if not specified)
				routeNamespace := gateway.Namespace
				if parentRef.Namespace != nil {
					routeNamespace = string(*parentRef.Namespace)
				}
				if routeNamespace == gateway.Namespace {
					return true, nil
				}
			}
		}
	}

	// Check for UDPRoutes
	udpRoutes, err := gatewayClient.GatewayV1alpha2().UDPRoutes(gateway.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return false, err
	}

	for _, route := range udpRoutes.Items {
		for _, parentRef := range route.Spec.ParentRefs {
			if parentRef.Name == gatewayv1.ObjectName(gateway.Name) {
				// Check if the namespace matches (default to same namespace if not specified)
				routeNamespace := gateway.Namespace
				if parentRef.Namespace != nil {
					routeNamespace = string(*parentRef.Namespace)
				}
				if routeNamespace == gateway.Namespace {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

func GetUnusedGateways(filterOpts *filters.Options, clientset kubernetes.Interface, gatewayClient gatewayclientset.Interface, outputFormat string, opts common.Opts) (string, error) {
	resources := make(map[string]map[string][]ResourceInfo)
	for _, namespace := range filterOpts.Namespaces(clientset) {
		diff, err := processNamespaceGateways(clientset, gatewayClient, namespace, filterOpts, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to process namespace %s: %v\n", namespace, err)
			continue
		}
		switch opts.GroupBy {
		case "namespace":
			resources[namespace] = make(map[string][]ResourceInfo)
			resources[namespace]["Gateway"] = diff
		case "resource":
			appendResources(resources, "Gateway", namespace, diff)
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

	unusedGateways, err := unusedResourceFormatter(outputFormat, outputBuffer, opts, jsonResponse)
	if err != nil {
		fmt.Printf("err: %v\n", err)
	}

	return unusedGateways, nil
}