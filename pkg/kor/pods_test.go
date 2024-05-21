package kor

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/yonahd/kor/pkg/filters"
)

func createTestPods(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	pod1 := CreateTestPod(testNamespace, "pod-1", "", nil, AppLabels)
	pod1.Status = corev1.PodStatus{
		Phase:   corev1.PodRunning,
		Reason:  "",
		Message: "",
	}
	pod2 := CreateTestPod(testNamespace, "pod-2", "", nil, AppLabels)
	pod2.Status = corev1.PodStatus{
		Phase:   corev1.PodFailed,
		Reason:  "Evicted",
		Message: "",
	}

	pod3 := CreateTestPod(testNamespace, "pod-3", "", nil, AppLabels)
	pod3.Status = corev1.PodStatus{
		Phase:   corev1.PodFailed,
		Reason:  "CrashLoopBackOff",
		Message: "",
	}

	pod4 := CreateTestPod(testNamespace, "pod-4", "", nil, AppLabels)
	pod4.Status = corev1.PodStatus{
		Phase:   corev1.PodSucceeded,
		Reason:  "",
		Message: "",
	}

	pod5 := CreateTestPod(testNamespace, "pod-5", "", nil, AppLabels)
	pod5.Labels = map[string]string{"kor/used": "true"}
	pod5.Status = corev1.PodStatus{
		Phase:   corev1.PodFailed,
		Reason:  "Evicted",
		Message: "",
	}

	pod6 := CreateTestPod(testNamespace, "pod-6", "", nil, AppLabels)
	pod6.Labels = map[string]string{"kor/used": "false"}
	pod6.Status = corev1.PodStatus{
		Phase:   corev1.PodFailed,
		Reason:  "Evicted",
		Message: "",
	}

	pods := []*corev1.Pod{
		pod1,
		pod2,
		pod3,
		pod4,
		pod5,
		pod6,
	}

	// Add test pods to the clientset
	for _, pod := range pods {
		_, err = clientset.CoreV1().Pods(pod.Namespace).Create(context.TODO(), pod, v1.CreateOptions{})
		if err != nil {
			t.Fatalf("Error creating fake pod: %v", err)
		}
	}

	return clientset
}

func TestProcessNamespacePods(t *testing.T) {
	clientset := createTestPods(t)
	evictedPods, err := processNamespacePods(clientset, testNamespace, &filters.Options{})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expectedEvictedPods := []string{
		"pod-2",
		"pod-6",
	}

	if len(evictedPods) != len(expectedEvictedPods) {
		t.Errorf("Expected %d evicted pods, got %d", len(expectedEvictedPods), len(evictedPods))
	}

	for i, pod := range evictedPods {
		if pod.Name != expectedEvictedPods[i] {
			t.Errorf("Expected evicted pod %s, got %s", expectedEvictedPods[i], pod)
		}
	}
}

func TestGetUnusedPodsStructured(t *testing.T) {
	clientset := createTestPods(t)

	opts := Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	output, err := GetUnusedPods(&filters.Options{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedPodsStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"Pod": {
				"pod-2",
				"pod-6",
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
	_ = corev1.AddToScheme(scheme.Scheme)
}
