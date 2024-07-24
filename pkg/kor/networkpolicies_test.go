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

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

func createTestNetworkPolicies(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	testNamespace2 := "another-namespace"
	namespaces := []string{testNamespace, testNamespace2}

	for _, ns := range namespaces {
		_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{Name: ns},
		}, v1.CreateOptions{})

		if err != nil {
			t.Fatalf("Error creating namespace %s: %v", ns, err)
		}
	}

	podLabels1 := map[string]string{
		"app.kubernetes.io/name":    "my-app",
		"app.kubernetes.io/version": "v1",
		"product.my-org/name":       "my-app",
	}
	podLabels2 := map[string]string{"app.kubernetes.io/version": "v2"}

	pods := []*corev1.Pod{
		CreateTestPod(testNamespace, "pod-1", "", nil, podLabels1),
		CreateTestPod(testNamespace, "pod-2", "", nil, AppLabels),
		CreateTestPod(testNamespace2, "pod-1", "", nil, podLabels2),
		CreateTestPod(testNamespace2, "pod-2", "", nil, AppLabels),
	}

	for _, pod := range pods {
		_, err := clientset.CoreV1().Pods(pod.Namespace).Create(context.TODO(), pod, v1.CreateOptions{})
		if err != nil {
			t.Fatalf("Error creating fake pod: %v", err)
		}
	}

	netpols := []*networkingv1.NetworkPolicy{
		// with kor labels
		CreateTestNetworkPolicy("netpol-1", testNamespace, UsedLabels, v1.LabelSelector{}, nil, nil),
		CreateTestNetworkPolicy("netpol-2", testNamespace, UnusedLabels, v1.LabelSelector{}, nil, nil),

		// with pod selectors
		// no pods are selected
		CreateTestNetworkPolicy("netpol-3", testNamespace, AppLabels, *v1.SetAsLabelSelector(podLabels2), nil, nil),

		// with ingress/egress rules
		// deny-all ingress
		CreateTestNetworkPolicy("netpol-4", testNamespace, AppLabels, *v1.SetAsLabelSelector(podLabels1), nil, nil),
		// allow-all ingress
		CreateTestNetworkPolicy("netpol-5", testNamespace, AppLabels, *v1.SetAsLabelSelector(podLabels1), []networkingv1.NetworkPolicyIngressRule{{}}, nil),
		// allow ingress to some pods
		CreateTestNetworkPolicy("netpol-6", testNamespace, AppLabels, *v1.SetAsLabelSelector(podLabels1), []networkingv1.NetworkPolicyIngressRule{{
			From: []networkingv1.NetworkPolicyPeer{
				{
					PodSelector: &v1.LabelSelector{
						MatchLabels: podLabels2,
					},
				},
			},
		}}, nil),
		// ingress matches 0 pods
		CreateTestNetworkPolicy("netpol-7", testNamespace, AppLabels, *v1.SetAsLabelSelector(podLabels1), []networkingv1.NetworkPolicyIngressRule{{
			From: []networkingv1.NetworkPolicyPeer{
				{
					PodSelector: &v1.LabelSelector{
						MatchLabels: map[string]string{"product.my-org/name": "unknown"},
					},
				},
			},
		}}, nil),
		// with ipBlock
		CreateTestNetworkPolicy("netpol-8", testNamespace, AppLabels, *v1.SetAsLabelSelector(podLabels1), []networkingv1.NetworkPolicyIngressRule{{
			From: []networkingv1.NetworkPolicyPeer{
				{
					IPBlock: &networkingv1.IPBlock{
						CIDR: "172.17.0.0/16",
					},
				},
			},
		}}, nil),
	}

	netpol9 := CreateTestNetworkPolicy("netpol-9", testNamespace, AppLabels, *v1.SetAsLabelSelector(podLabels1), nil, []networkingv1.NetworkPolicyEgressRule{{
		To: []networkingv1.NetworkPolicyPeer{
			{
				PodSelector: &v1.LabelSelector{
					MatchLabels: map[string]string{"product.my-org/name": "unknown"},
				},
			},
		},
	}}) // egress only - matches 0 pods
	netpol9.Spec.PolicyTypes = []networkingv1.PolicyType{networkingv1.PolicyTypeEgress}
	netpols = append(netpols, netpol9)

	for _, netpol := range netpols {
		_, err := clientset.NetworkingV1().NetworkPolicies(netpol.Namespace).Create(context.TODO(), netpol, v1.CreateOptions{})
		if err != nil {
			t.Fatalf("Error creating fake networkpolicy: %v", err)
		}
	}

	return clientset
}

func TestRetrievePodsForSelector(t *testing.T) {
	clientset := createTestNetworkPolicies(t)

	selector := &v1.LabelSelector{
		MatchLabels: map[string]string{
			"app.kubernetes.io/version": "v1",
		},
	}
	pods, err := retrievePodsForSelector(clientset, testNamespace, selector)
	if err != nil {
		t.Errorf("Error retrieving pods for selector %v: %v", selector, err)
	}

	expectedPods := []string{
		"pod-1",
	}

	if len(pods) != len(expectedPods) {
		t.Errorf("Expected %d pods, got %d", len(expectedPods), len(pods))
	}

	for i, pod := range pods {
		if pod.Name != expectedPods[i] {
			t.Errorf("Expected pod %s, got %v", expectedPods[i], pod)
		}
	}
}

func TestIsAnyPodMatchedInSources(t *testing.T) {
	clientset := createTestNetworkPolicies(t)

	sources := []networkingv1.NetworkPolicyPeer{
		{
			PodSelector: &v1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/version": "v1",
				},
			},
		},
	}

	matched, err := isAnyPodMatchedInSources(clientset, sources)
	if err != nil {
		t.Errorf("Error checking if sources match any pods: %v", err)
	}

	if !matched {
		t.Error("Expected matching pods, got none")
	}
}

func TestIsAnyIngressRuleUsed(t *testing.T) {
	clientset := createTestNetworkPolicies(t)

	netpol := CreateTestNetworkPolicy("netpol-0", testNamespace, AppLabels, v1.LabelSelector{}, nil, nil)

	used, err := isAnyIngressRuleUsed(clientset, *netpol)
	if err != nil {
		t.Errorf("Error checking if any ingress rule is used: %v", err)
	}

	if !used {
		t.Error("Expected ingress rules in use, got none")
	}
}

func TestIsAnyEgressRuleUsed(t *testing.T) {
	clientset := createTestNetworkPolicies(t)

	netpol := CreateTestNetworkPolicy("netpol-0", testNamespace, AppLabels, v1.LabelSelector{}, nil, nil)

	used, err := isAnyEgressRuleUsed(clientset, *netpol)
	if err != nil {
		t.Errorf("Error checking if any egress rule is used: %v", err)
	}

	if used {
		t.Error("Expected ingress rules not in use, got used rules")
	}
}

func TestProcessNamespaceNetworkPolicies(t *testing.T) {
	clientset := createTestNetworkPolicies(t)

	unusedNetpols, err := processNamespaceNetworkPolicies(clientset, testNamespace, &filters.Options{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	expectedUnusedNetpols := []string{
		"netpol-2",
		"netpol-3",
		"netpol-7",
		"netpol-9",
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

	opts := common.Opts{
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
				"netpol-2",
				"netpol-3",
				"netpol-7",
				"netpol-9",
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
