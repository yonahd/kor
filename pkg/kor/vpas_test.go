package kor

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	discoveryfake "k8s.io/client-go/discovery/fake"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

func createTestVpas(t *testing.T) (*fake.Clientset, *dynamicfake.FakeDynamicClient) {
	clientset := fake.NewClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	deploymentName := "test-deployment"
	statefulSetName := "test-statefulset"

	deployment1 := CreateTestDeployment(testNamespace, deploymentName, 1, AppLabels)
	_, err = clientset.AppsV1().Deployments(testNamespace).Create(context.TODO(), deployment1, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake deployment: %v", err)
	}

	statefulSet1 := CreateTestStatefulSet(testNamespace, statefulSetName, 1, AppLabels)
	_, err = clientset.AppsV1().StatefulSets(testNamespace).Create(context.TODO(), statefulSet1, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake statefulset: %v", err)
	}

	scheme := runtime.NewScheme()
	gvrToListKind := map[schema.GroupVersionResource]string{
		VpaGVR: "VerticalPodAutoscalerList",
	}
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind)

	objects := []*unstructured.Unstructured{
		CreateTestVpa(testNamespace, "test-vpa1", "Deployment", deploymentName, AppLabels),
		CreateTestVpa(testNamespace, "test-vpa2", "Deployment", "non-existing-deployment", AppLabels),
		CreateTestVpa(testNamespace, "test-vpa3", "Deployment", deploymentName, UsedLabels),
		CreateTestVpa(testNamespace, "test-vpa4", "Deployment", "non-existing-deployment", UnusedLabels),
		CreateTestVpa(testNamespace, "test-vpa5", "StatefulSet", statefulSetName, AppLabels),
		CreateTestVpa(testNamespace, "test-vpa6", "StatefulSet", "non-existing-statefulset", AppLabels),
	}
	for _, vpa := range objects {
		_, err = dynamicClient.Resource(VpaGVR).Namespace(testNamespace).Create(context.TODO(), vpa, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("Error creating fake Vpa: %v", err)
		}
	}

	return clientset, dynamicClient
}

func TestExtractUnusedVpas(t *testing.T) {
	clientset, dynamicClient := createTestVpas(t)

	unusedVpas, err := processNamespaceVpas(clientset, dynamicClient, testNamespace, &filters.Options{}, common.Opts{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(unusedVpas) != 3 {
		t.Errorf("Expected 3 unused VPAs, got %d", len(unusedVpas))
	}

	unusedNames := map[string]bool{}
	for _, vpa := range unusedVpas {
		unusedNames[vpa.Name] = true
	}
	for _, name := range []string{"test-vpa2", "test-vpa4", "test-vpa6"} {
		if !unusedNames[name] {
			t.Errorf("Expected %s in unused VPAs, got %v", name, unusedNames)
		}
	}
}

func TestProcessNamespaceVpasWithDeleteFlag(t *testing.T) {
	clientset, dynamicClient := createTestVpas(t)

	opts := common.Opts{
		DeleteFlag:    true,
		NoInteractive: true,
	}
	unusedVpas, err := processNamespaceVpas(clientset, dynamicClient, testNamespace, &filters.Options{}, opts)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(unusedVpas) != 3 {
		t.Errorf("Expected 3 unused VPAs after deletion, got %d", len(unusedVpas))
	}

	remainingVpas, err := dynamicClient.Resource(VpaGVR).Namespace(testNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("Error listing VPAs: %v", err)
	}

	if len(remainingVpas.Items) != 3 {
		t.Errorf("Expected 3 remaining VPAs, got %d", len(remainingVpas.Items))
	}
}

func TestGetUnusedVpasStructured(t *testing.T) {
	clientset, dynamicClient := createTestVpas(t)

	clientset.Discovery().(*discoveryfake.FakeDiscovery).Resources = append(
		clientset.Discovery().(*discoveryfake.FakeDiscovery).Resources,
		&metav1.APIResourceList{
			GroupVersion: "autoscaling.k8s.io/v1",
			APIResources: []metav1.APIResource{
				{
					Name:       "verticalpodautoscalers",
					Namespaced: true,
					Kind:       "VerticalPodAutoscaler",
				},
			},
		},
	)

	opts := common.Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	output, err := GetUnusedVpas(&filters.Options{}, clientset, dynamicClient, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedVpas: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"Vpa": {
				"test-vpa2",
				"test-vpa4",
				"test-vpa6",
			},
		},
	}

	var actualOutput map[string]map[string][]string
	if err := json.Unmarshal([]byte(output), &actualOutput); err != nil {
		t.Fatalf("Error unmarshaling actual output: %v", err)
	}

	if !reflect.DeepEqual(expectedOutput, actualOutput) {
		t.Errorf("Expected output does not match actual output: %v", actualOutput)
	}
}

func TestGetUnusedVpasUnsupported(t *testing.T) {
	clientset, dynamicClient := createTestVpas(t)

	opts := common.Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	output, err := GetUnusedVpas(&filters.Options{}, clientset, dynamicClient, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedVpas: %v", err)
	}

	if output != "{}" {
		t.Errorf("Expected '{}' when VPA is unsupported, got %s", output)
	}
}

func TestGetVpaScaleTargetRef(t *testing.T) {
	vpaNoTargetRef := CreateTestVpa("test-namespace", "vpa-no-targetref", "Deployment", "non-existing-deployment", AppLabels)
	unstructured.RemoveNestedField(vpaNoTargetRef.Object, "spec", "targetRef")

	vpaNoKind := CreateTestVpa("test-namespace", "vpa-no-kind", "Deployment", "non-existing-deployment", AppLabels)
	if err := unstructured.SetNestedField(vpaNoKind.Object, "", "spec", "targetRef", "kind"); err != nil {
		t.Fatalf("Error clearing kind from VPA: %v", err)
	}

	vpaNoName := CreateTestVpa("test-namespace", "vpa-no-name", "Deployment", "", AppLabels)

	tests := []struct {
		name     string
		vpa      *unstructured.Unstructured
		wantKind string
		wantName string
		wantErr  bool
	}{
		{
			name:     "valid target ref",
			vpa:      CreateTestVpa("test-namespace", "vpa-valid", "Deployment", "test-deployment", AppLabels),
			wantKind: "Deployment",
			wantName: "test-deployment",
		},
		{
			name:    "missing target ref",
			vpa:     vpaNoTargetRef,
			wantErr: true,
		},
		{
			name:    "missing kind",
			vpa:     vpaNoKind,
			wantErr: true,
		},
		{
			name:    "missing name",
			vpa:     vpaNoName,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kind, name, err := getVpaScaleTargetRef(tt.vpa)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if kind != tt.wantKind || name != tt.wantName {
				t.Errorf("expected kind=%q name=%q, got kind=%q name=%q", tt.wantKind, tt.wantName, kind, name)
			}
		})
	}
}

func TestProcessNamespaceVpasWithExcludeLabels(t *testing.T) {
	clientset, dynamicClient := createTestVpas(t)

	filterOpts := &filters.Options{
		ExcludeLabels: []string{"kor/used=false"},
	}
	unusedVpas, err := processNamespaceVpas(clientset, dynamicClient, testNamespace, filterOpts, common.Opts{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(unusedVpas) != 2 {
		t.Fatalf("Expected 2 unused VPAs, got %d", len(unusedVpas))
	}

	unusedNames := map[string]bool{}
	for _, vpa := range unusedVpas {
		unusedNames[vpa.Name] = true
	}
	for _, name := range []string{"test-vpa2", "test-vpa6"} {
		if !unusedNames[name] {
			t.Errorf("Expected %s in unused VPAs, got %v", name, unusedNames)
		}
	}
	if unusedNames["test-vpa4"] {
		t.Errorf("Expected test-vpa4 to be excluded by the label filter, got %v", unusedNames)
	}
}

func TestProcessNamespaceVpasSkipsInvalidTargetRef(t *testing.T) {
	clientset, dynamicClient := createTestVpas(t)

	vpaNoTargetRef := CreateTestVpa(testNamespace, "test-vpa-no-targetref", "Deployment", "non-existing-deployment", AppLabels)
	unstructured.RemoveNestedField(vpaNoTargetRef.Object, "spec", "targetRef")

	vpaNoKind := CreateTestVpa(testNamespace, "test-vpa-no-kind", "Deployment", "non-existing-deployment", AppLabels)
	if err := unstructured.SetNestedField(vpaNoKind.Object, "", "spec", "targetRef", "kind"); err != nil {
		t.Fatalf("Error clearing kind from VPA: %v", err)
	}

	vpaDaemonSet := CreateTestVpa(testNamespace, "test-vpa-daemonset", "DaemonSet", "non-existing-daemonset", AppLabels)

	for _, vpa := range []*unstructured.Unstructured{vpaNoTargetRef, vpaNoKind, vpaDaemonSet} {
		if _, err := dynamicClient.Resource(VpaGVR).Namespace(testNamespace).Create(context.TODO(), vpa, metav1.CreateOptions{}); err != nil {
			t.Fatalf("Error creating fake Vpa: %v", err)
		}
	}

	unusedVpas, err := processNamespaceVpas(clientset, dynamicClient, testNamespace, &filters.Options{}, common.Opts{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(unusedVpas) != 3 {
		t.Errorf("Expected 3 unused VPAs, got %d", len(unusedVpas))
	}
	for _, vpa := range unusedVpas {
		switch vpa.Name {
		case "test-vpa-no-targetref", "test-vpa-no-kind", "test-vpa-daemonset":
			t.Errorf("VPA %s should not be reported as unused", vpa.Name)
		}
	}
}

func TestProcessNamespaceVpasMarkedUnusedReason(t *testing.T) {
	clientset, dynamicClient := createTestVpas(t)

	unusedVpas, err := processNamespaceVpas(clientset, dynamicClient, testNamespace, &filters.Options{}, common.Opts{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	target := "test-vpa4"
	for _, vpa := range unusedVpas {
		if vpa.Name == target {
			if vpa.Reason != "Marked with unused label" {
				t.Errorf("Expected reason %q for %s, got %q", "Marked with unused label", target, vpa.Reason)
			}
			return
		}
	}
	t.Errorf("Expected %s in unused VPAs, got %v", target, unusedVpas)
}

func TestGetUnusedVpasYamlOutput(t *testing.T) {
	clientset, dynamicClient := createTestVpas(t)

	clientset.Discovery().(*discoveryfake.FakeDiscovery).Resources = append(
		clientset.Discovery().(*discoveryfake.FakeDiscovery).Resources,
		&metav1.APIResourceList{
			GroupVersion: "autoscaling.k8s.io/v1",
			APIResources: []metav1.APIResource{
				{
					Name:       "verticalpodautoscalers",
					Namespaced: true,
					Kind:       "VerticalPodAutoscaler",
				},
			},
		},
	)

	opts := common.Opts{
		GroupBy: "namespace",
	}
	output, err := GetUnusedVpas(&filters.Options{}, clientset, dynamicClient, "yaml", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedVpas: %v", err)
	}

	for _, name := range []string{"test-vpa2", "test-vpa4", "test-vpa6"} {
		if !strings.Contains(output, name) {
			t.Errorf("Expected %s in yaml output, got:\n%s", name, output)
		}
	}
}

func TestGetUnusedVpasGroupByResource(t *testing.T) {
	clientset, dynamicClient := createTestVpas(t)

	clientset.Discovery().(*discoveryfake.FakeDiscovery).Resources = append(
		clientset.Discovery().(*discoveryfake.FakeDiscovery).Resources,
		&metav1.APIResourceList{
			GroupVersion: "autoscaling.k8s.io/v1",
			APIResources: []metav1.APIResource{
				{
					Name:       "verticalpodautoscalers",
					Namespaced: true,
					Kind:       "VerticalPodAutoscaler",
				},
			},
		},
	)

	opts := common.Opts{
		GroupBy: "resource",
	}
	output, err := GetUnusedVpas(&filters.Options{}, clientset, dynamicClient, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedVpas: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		"Vpa": {
			testNamespace: {
				"test-vpa2",
				"test-vpa4",
				"test-vpa6",
			},
		},
	}

	var actualOutput map[string]map[string][]string
	if err := json.Unmarshal([]byte(output), &actualOutput); err != nil {
		t.Fatalf("Error unmarshaling actual output: %v", err)
	}

	if !reflect.DeepEqual(expectedOutput, actualOutput) {
		t.Errorf("Expected output does not match actual output: %v", actualOutput)
	}
}
