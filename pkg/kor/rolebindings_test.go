package kor

import (
	"context"
	"testing"

	"github.com/yonahd/kor/pkg/filters"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
)

func createTestRoleBindings(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	testRoleBindings1 := CreateTestRoleBinding(
		testNamespace,
		"role-ref-rb",
		"test-rb-sa",
		&rbacv1.RoleRef{
			Kind: "Role",
			Name: "empty-role",
		})
	_, err = clientset.RbacV1().RoleBindings(testNamespace).Create(context.TODO(), testRoleBindings1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "RoleBinding: testRoleBindings1", err)
	}

	testRoleBindings2 := CreateTestRoleBinding(
		testNamespace,
		"cluster-role-ref-rb",
		"test-rb-sa",
		&rbacv1.RoleRef{
			Kind: "ClusterRole",
			Name: "empty-cluster-rule",
		})
	_, err = clientset.RbacV1().RoleBindings(testNamespace).Create(context.TODO(), testRoleBindings2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "RoleBinding: testRoleBindings2", err)
	}

	return clientset
}

func TestProcessNamespaceRoleBindings(t *testing.T) {
	clientset := createTestRoleBindings(t)

	unusedRoleBindings, err := processNamespaceRoleBindings(clientset, testNamespace, &filters.Options{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(unusedRoleBindings) != 2 {
		t.Errorf("Expected 2 unused role bindings, got %d", len(unusedRoleBindings))
	}
}

func init() {
	scheme.Scheme = runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme.Scheme)
}
