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

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
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

	opts := common.Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	output, err := GetUnusedReplicaSets(&filters.Options{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedReplicaSetsStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"ReplicaSet": {"test-replicaSet2"},
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

func TestFilterDeploymentOwnedReplicaSets(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	// Create a deployment
	deployment := CreateTestDeployment(testNamespace, "test-deployment", 1, AppLabels)
	_, err = clientset.AppsV1().Deployments(testNamespace).Create(context.TODO(), deployment, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake deployment: %v", err)
	}

	// Create two replicasets - one owned by deployment, one standalone
	var count int32 = 0
	
	// ReplicaSet owned by deployment
	ownedRS := CreateTestReplicaSet(testNamespace, "owned-rs", &count, &appsv1.ReplicaSetStatus{
		Replicas:             count,
		AvailableReplicas:    count,
		ReadyReplicas:        count,
		FullyLabeledReplicas: count,
	})
	// Add owner reference to deployment
	ownedRS.OwnerReferences = []v1.OwnerReference{
		{
			Kind: "Deployment",
			Name: "test-deployment",
		},
	}
	
	// Standalone ReplicaSet
	standaloneRS := CreateTestReplicaSet(testNamespace, "standalone-rs", &count, &appsv1.ReplicaSetStatus{
		Replicas:             count,
		AvailableReplicas:    count,
		ReadyReplicas:        count,
		FullyLabeledReplicas: count,
	})

	_, err = clientset.AppsV1().ReplicaSets(testNamespace).Create(context.TODO(), ownedRS, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake replicaSet: %v", err)
	}

	_, err = clientset.AppsV1().ReplicaSets(testNamespace).Create(context.TODO(), standaloneRS, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake replicaSet: %v", err)
	}

	// Test without filter - should return both
	filterOptsNoSkip := &filters.Options{IgnoreOwnerReferences: false}
	unusedWithoutFilter, err := processNamespaceReplicaSets(clientset, testNamespace, filterOptsNoSkip, common.Opts{})
	if err != nil {
		t.Fatalf("Error retrieving unused replica sets: %v", err)
	}

	if len(unusedWithoutFilter) != 2 {
		t.Errorf("Expected 2 unused ReplicaSet objects without filter, got %d", len(unusedWithoutFilter))
	}

	// Test with filter - should return only standalone
	filterOptsWithSkip := &filters.Options{IgnoreOwnerReferences: true}
	unusedWithFilter, err := processNamespaceReplicaSets(clientset, testNamespace, filterOptsWithSkip, common.Opts{})
	if err != nil {
		t.Fatalf("Error retrieving unused replica sets: %v", err)
	}

	if len(unusedWithFilter) != 1 {
		t.Errorf("Expected 1 unused ReplicaSet object with filter, got %d", len(unusedWithFilter))
	}

	if unusedWithFilter[0].Name != "standalone-rs" {
		t.Errorf("Expected standalone-rs to be unused, got %s", unusedWithFilter[0].Name)
	}
}

func init() {
	scheme.Scheme = runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme.Scheme)
}
