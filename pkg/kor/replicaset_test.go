package kor

import (
	"context"
	"encoding/json"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"reflect"
	"testing"
)

func createTestReplicaSets(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	var count1 int32 = 1
	replicaSet1 := CreateTestReplicaSet(testNamespace, "test-replicaSet1", &count1, &appsv1.ReplicaSetStatus{
		Replicas:             count1,
		AvailableReplicas:    count1,
		ReadyReplicas:        count1,
		FullyLabeledReplicas: count1,
	})
	var count2 int32 = 0
	replicaSet2 := CreateTestReplicaSet(testNamespace, "test-replicaSet2", &count2, &appsv1.ReplicaSetStatus{
		Replicas:             count2,
		AvailableReplicas:    count2,
		ReadyReplicas:        count2,
		FullyLabeledReplicas: count2,
	})

	_, err = clientset.AppsV1().ReplicaSets(testNamespace).Create(context.TODO(), replicaSet1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake replicaSet: %v", err)
	}

	_, err = clientset.AppsV1().ReplicaSets(testNamespace).Create(context.TODO(), replicaSet2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake replicaSet: %v", err)
	}
	return clientset
}

func TestProcessNamespaceReplicaSets(t *testing.T) {
	clientset := createTestReplicaSets(t)

	includeExcludeLists := IncludeExcludeLists{
		IncludeListStr: "",
		ExcludeListStr: "",
	}

	opts := Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
	}

	output, err := GetUnusedReplicaSets(includeExcludeLists, &FilterOptions{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedReplicaSetsStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"ReplicaSets": {"test-replicaSet2"},
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

func init() {
	scheme.Scheme = runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme.Scheme)
}
