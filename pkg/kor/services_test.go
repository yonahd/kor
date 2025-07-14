package kor

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

func createTestServices(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	endpoint1 := CreateTestEndpoint(testNamespace, "test-endpoint1", 0, AppLabels)
	_, err = clientset.DiscoveryV1().EndpointSlices(testNamespace).Create(context.TODO(), endpoint1, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating fake endpoint: %v", err)
	}

	endpoint2 := CreateTestEndpoint(testNamespace, "test-endpoint2", 1, AppLabels)
	_, err = clientset.DiscoveryV1().EndpointSlices(testNamespace).Create(context.TODO(), endpoint2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake endpoint: %v", err)
	}

	endpoint3 := CreateTestEndpoint(testNamespace, "test-endpoint3", 1, UsedLabels)
	_, err = clientset.DiscoveryV1().EndpointSlices(testNamespace).Create(context.TODO(), endpoint3, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake endpoint: %v", err)
	}

	endpoint4 := CreateTestEndpoint(testNamespace, "test-endpoint4", 1, UnusedLabels)
	_, err = clientset.DiscoveryV1().EndpointSlices(testNamespace).Create(context.TODO(), endpoint4, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake endpoint: %v", err)
	}

	return clientset
}

func TestGetEndpointsWithoutSubsets(t *testing.T) {
	clientset := createTestServices(t)

	servicesWithoutEndpoints, err := processNamespaceServices(clientset, testNamespace, &filters.Options{}, common.Opts{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(servicesWithoutEndpoints) != 2 {
		t.Errorf("Expected 2 service without endpoint, got %d", len(servicesWithoutEndpoints))
	}

	if servicesWithoutEndpoints[0].Name != "test-endpoint1" || servicesWithoutEndpoints[1].Name != "test-endpoint4" {
		t.Errorf("Expected 'test-endpoint1' and 'test-endpoint4', got %s, %s", servicesWithoutEndpoints[0], servicesWithoutEndpoints[1])
	}
}

func TestGetUnusedServicesStructured(t *testing.T) {
	clientset := createTestServices(t)

	opts := common.Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	output, err := GetUnusedServices(&filters.Options{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedServicesStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"Service": {
				"test-endpoint1",
				"test-endpoint4",
			},
		},
	}

	var actualOutput map[string]map[string][]string
	if err := json.Unmarshal([]byte(output), &actualOutput); err != nil {
		t.Fatalf("Error unmarshaling actual output: %v", err)
	}

	if !reflect.DeepEqual(expectedOutput, actualOutput) {
		t.Errorf("Expected output does not match actual output")
	}
}

func TestFilterOwnerReferencedServices(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	// Create two services - one owned by deployment, one standalone
	// Service owned by deployment
	ownedService := CreateTestService(testNamespace, "owned-service")
	// Add owner reference to deployment
	ownedService.OwnerReferences = []v1.OwnerReference{
		{
			Kind: "Deployment",
			Name: "test-deployment",
		},
	}

	// Standalone Service
	standaloneService := CreateTestService(testNamespace, "standalone-service")

	_, err = clientset.CoreV1().Services(testNamespace).Create(context.TODO(), ownedService, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake service: %v", err)
	}

	_, err = clientset.CoreV1().Services(testNamespace).Create(context.TODO(), standaloneService, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake service: %v", err)
	}

	// Create EndpointSlices for the services
	// EndpointSlice for owned service (with endpoints)
	ownedEndpointSlice := CreateTestEndpoint(testNamespace, "owned-service-endpoints", 1, AppLabels)
	ownedEndpointSlice.Labels["kubernetes.io/service-name"] = "owned-service"
	ownedEndpointSlice.OwnerReferences = []v1.OwnerReference{
		{
			Kind: "Deployment",
			Name: "test-deployment",
		},
	}

	// EndpointSlice for standalone service (with endpoints)
	standaloneEndpointSlice := CreateTestEndpoint(testNamespace, "standalone-service-endpoints", 1, AppLabels)
	standaloneEndpointSlice.Labels["kubernetes.io/service-name"] = "standalone-service"

	_, err = clientset.DiscoveryV1().EndpointSlices(testNamespace).Create(context.TODO(), ownedEndpointSlice, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake endpoint slice: %v", err)
	}

	_, err = clientset.DiscoveryV1().EndpointSlices(testNamespace).Create(context.TODO(), standaloneEndpointSlice, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake endpoint slice: %v", err)
	}

	// Test without filter - should return both
	filterOptsNoSkip := &filters.Options{IgnoreOwnerReferences: false}
	unusedWithoutFilter, err := processNamespaceServices(clientset, testNamespace, filterOptsNoSkip, common.Opts{})
	if err != nil {
		t.Fatalf("Error retrieving unused services: %v", err)
	}

	if len(unusedWithoutFilter) != 0 {
		t.Errorf("Expected 0 unused Service objects without filter (both have endpoints), got %d", len(unusedWithoutFilter))
	}

	// Create EndpointSlices without endpoints to make them unused
	// EndpointSlice for owned service (without endpoints)
	ownedEndpointSliceNoEndpoints := CreateTestEndpoint(testNamespace, "owned-service-endpoints-empty", 0, AppLabels)
	ownedEndpointSliceNoEndpoints.Labels["kubernetes.io/service-name"] = "owned-service"
	ownedEndpointSliceNoEndpoints.OwnerReferences = []v1.OwnerReference{
		{
			Kind: "Deployment",
			Name: "test-deployment",
		},
	}

	// EndpointSlice for standalone service (without endpoints)
	standaloneEndpointSliceNoEndpoints := CreateTestEndpoint(testNamespace, "standalone-service-endpoints-empty", 0, AppLabels)
	standaloneEndpointSliceNoEndpoints.Labels["kubernetes.io/service-name"] = "standalone-service"

	_, err = clientset.DiscoveryV1().EndpointSlices(testNamespace).Create(context.TODO(), ownedEndpointSliceNoEndpoints, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake endpoint slice: %v", err)
	}

	_, err = clientset.DiscoveryV1().EndpointSlices(testNamespace).Create(context.TODO(), standaloneEndpointSliceNoEndpoints, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake endpoint slice: %v", err)
	}

	// Test without filter - should return both
	unusedWithoutFilter2, err := processNamespaceServices(clientset, testNamespace, filterOptsNoSkip, common.Opts{})
	if err != nil {
		t.Fatalf("Error retrieving unused services: %v", err)
	}

	if len(unusedWithoutFilter2) != 2 {
		t.Errorf("Expected 2 unused Service objects without filter, got %d", len(unusedWithoutFilter2))
	}

	// Test with filter - should return only standalone
	filterOptsWithSkip := &filters.Options{IgnoreOwnerReferences: true}
	unusedWithFilter, err := processNamespaceServices(clientset, testNamespace, filterOptsWithSkip, common.Opts{})
	if err != nil {
		t.Fatalf("Error retrieving unused services: %v", err)
	}

	if len(unusedWithFilter) != 1 {
		t.Errorf("Expected 1 unused Service object with filter, got %d", len(unusedWithFilter))
	}

	if unusedWithFilter[0].Name != "standalone-service" {
		t.Errorf("Expected standalone-service to be unused, got %s", unusedWithFilter[0].Name)
	}
}

func init() {
	scheme.Scheme = runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme.Scheme)
}
