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

func createTestDaemonSets(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	appLabels := map[string]string{}

	ds1 := CreateTestDaemonSet(testNamespace, "test-ds1", appLabels, &appsv1.DaemonSetStatus{
		CurrentNumberScheduled: 0,
	})
	ds2 := CreateTestDaemonSet(testNamespace, "test-ds2", appLabels, &appsv1.DaemonSetStatus{
		CurrentNumberScheduled: 1,
	})

	_, err = clientset.AppsV1().DaemonSets(testNamespace).Create(context.TODO(), ds1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "DaemonSet", err)
	}

	_, err = clientset.AppsV1().DaemonSets(testNamespace).Create(context.TODO(), ds2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "DaemonSet", err)
	}

	return clientset
}

func TestProcessNamespaceDaemonSets(t *testing.T) {
	clientset := createTestDaemonSets(t)

	daemonSetsWithoutReplicas, err := ProcessNamespaceDaemonSets(clientset, testNamespace, &filters.Options{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(daemonSetsWithoutReplicas) != 1 {
		t.Errorf("Expected 1 DaemonSet without replicas, got %d", len(daemonSetsWithoutReplicas))
	}

	if daemonSetsWithoutReplicas[0] != "test-ds1" {
		t.Errorf("Expected 'test-ds1', got %s", daemonSetsWithoutReplicas[0])
	}
}

func TestGetUnusedDaemonSetsStructured(t *testing.T) {
	clientset := createTestDaemonSets(t)

	opts := Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
	}

	output, err := GetUnusedDaemonSets(&filters.Options{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedDaemonSetsStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"DaemonSets": {"test-ds1"},
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
