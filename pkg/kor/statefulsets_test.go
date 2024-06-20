package kor

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/yonahd/kor/pkg/filters"
)

func createTestStatefulSets(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	sts1 := CreateTestStatefulSet(testNamespace, "test-sts1", 0, AppLabels)
	_, err = clientset.AppsV1().StatefulSets(testNamespace).Create(context.TODO(), sts1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "statefulSet", err)
	}

	sts2 := CreateTestStatefulSet(testNamespace, "test-sts2", 1, AppLabels)
	_, err = clientset.AppsV1().StatefulSets(testNamespace).Create(context.TODO(), sts2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "statefulSet", err)
	}

	sts3 := CreateTestStatefulSet(testNamespace, "test-sts3", 1, UsedLabels)
	_, err = clientset.AppsV1().StatefulSets(testNamespace).Create(context.TODO(), sts3, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "statefulSet", err)
	}

	sts4 := CreateTestStatefulSet(testNamespace, "test-sts4", 1, UnusedLabels)
	_, err = clientset.AppsV1().StatefulSets(testNamespace).Create(context.TODO(), sts4, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "statefulSet", err)
	}

	return clientset
}

func TestProcessNamespaceStatefulSets(t *testing.T) {
	clientset := createTestStatefulSets(t)

	statefulSetsWithoutReplicas, err := processNamespaceStatefulSets(clientset, testNamespace, &filters.Options{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(statefulSetsWithoutReplicas) != 2 {
		t.Errorf("Expected 2 deployment without replicas, got %d", len(statefulSetsWithoutReplicas))
	}

	if statefulSetsWithoutReplicas[0].Name != "test-sts1" || statefulSetsWithoutReplicas[1].Name != "test-sts4" {
		t.Errorf("Expected 'test-sts1' and 'test-sts4, got %s, %s", statefulSetsWithoutReplicas[0], statefulSetsWithoutReplicas[1])
	}
}

func TestGetUnusedStatefulSetsStructured(t *testing.T) {
	clientset := createTestStatefulSets(t)

	opts := Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	output, err := GetUnusedStatefulSets(&filters.Options{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedStatefulSetsStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"StatefulSet": {
				"test-sts1",
				"test-sts4",
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

func init() {
	scheme.Scheme = runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme.Scheme)
}
