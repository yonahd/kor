package kor

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/yonahd/kor/pkg/filters"
)

func createTestNetworkPolicies(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	podLabels := map[string]string{
		"app.kubernetes.io/name":    "my-app",
		"app.kubernetes.io/version": "v1",
		"product.my-org/name":       "my-app",
	}
	noMatchLabels := map[string]string{"app.kubernetes.io/version": "v2"}

	pods := []*corev1.Pod{
		CreateTestPod(testNamespace, "pod-1", "", nil, podLabels),
		CreateTestPod(testNamespace, "pod-2", "", nil, AppLabels),
	}

	for _, pod := range pods {
		_, err = clientset.CoreV1().Pods(testNamespace).Create(context.TODO(), pod, v1.CreateOptions{})
		if err != nil {
			t.Fatalf("Error creating fake pod: %v", err)
		}
	}

	netpols := []*networkingv1.NetworkPolicy{
		// all pods are selected
		CreateTestNetworkPolicy("netpol-1", testNamespace, v1.LabelSelector{}, AppLabels),
		CreateTestNetworkPolicy("netpol-2", testNamespace, v1.LabelSelector{}, UsedLabels),
		CreateTestNetworkPolicy("netpol-3", testNamespace, v1.LabelSelector{}, UnusedLabels),
		// some pods are selected
		CreateTestNetworkPolicy("netpol-4", testNamespace, *v1.SetAsLabelSelector(podLabels), AppLabels),
		CreateTestNetworkPolicy("netpol-5", testNamespace, *v1.SetAsLabelSelector(podLabels), UnusedLabels),
		CreateTestNetworkPolicy("netpol-6", testNamespace, *v1.SetAsLabelSelector(podLabels), UsedLabels),
		// no pods are selected
		CreateTestNetworkPolicy("netpol-7", testNamespace, *v1.SetAsLabelSelector(noMatchLabels), AppLabels),
		CreateTestNetworkPolicy("netpol-8", testNamespace, *v1.SetAsLabelSelector(noMatchLabels), UnusedLabels),
		CreateTestNetworkPolicy("netpol-9", testNamespace, *v1.SetAsLabelSelector(noMatchLabels), UsedLabels),
	}

	for _, netpol := range netpols {
		_, err = clientset.NetworkingV1().NetworkPolicies(netpol.Namespace).Create(context.TODO(), netpol, v1.CreateOptions{})
		if err != nil {
			t.Fatalf("Error creating fake networkpolicy: %v", err)
		}
	}

	return clientset
}

func TestProcessNamespaceNetworkPolicies(t *testing.T) {
	clientset := createTestNetworkPolicies(t)

	unusedNetpols, err := processNamespaceNetworkPolicies(clientset, testNamespace, &filters.Options{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	expectedUnusedNetpols := []string{
		"netpol-3",
		"netpol-5",
		"netpol-7",
		"netpol-8",
	}

	if len(unusedNetpols) != len(expectedUnusedNetpols) {
		t.Errorf("Expected %d unused networkpolicies, got %d", len(expectedUnusedNetpols), len(unusedNetpols))
	}

	for i, netpol := range unusedNetpols {
		if netpol.Name != expectedUnusedNetpols[i] {
			t.Errorf("Expected unused networkpolicy %s, got %s", expectedUnusedNetpols[i], netpol)
		}
	}
}

func TestGetUnusedNetworkPolicies(t *testing.T) {
	clientset := createTestNetworkPolicies(t)

	opts := Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	output, err := GetUnusedNetworkPolicies(&filters.Options{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedNetworkPolicies: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"NetworkPolicy": []string{
				"netpol-3",
				"netpol-5",
				"netpol-7",
				"netpol-8",
			},
		},
	}

	var actualOutput map[string]map[string][]string
	if err := json.Unmarshal([]byte(output), &actualOutput); err != nil {
		t.Fatalf("Error unmarshaling actual output: %v", err)
	}

	if !reflect.DeepEqual(expectedOutput, actualOutput) {
		t.Errorf("Expected output does not match actual output")
		t.Errorf("Expected: %v", expectedOutput)
		t.Errorf("Actual: %v", actualOutput)
	}
}

func init() {
	scheme.Scheme = runtime.NewScheme()
	_ = networkingv1.AddToScheme(scheme.Scheme)
}
