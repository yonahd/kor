package kor

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

func createTestRoleBindings(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	rb1 := CreateTestRoleBinding(
		testNamespace,
		"test-rb1",
		"sa1",
		&rbacv1.RoleRef{
			Kind: "Role",
			Name: "non-exists-role",
		})
	_, err = clientset.RbacV1().RoleBindings(testNamespace).Create(context.TODO(), rb1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "RoleBinding: rb1", err)
	}

	rb2 := CreateTestRoleBinding(
		testNamespace,
		"test-rb2",
		"sa2",
		&rbacv1.RoleRef{
			Kind: "ClusterRole",
			Name: "non-existing-cluster-rule",
		})
	_, err = clientset.RbacV1().RoleBindings(testNamespace).Create(context.TODO(), rb2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "RoleBinding: rb2", err)
	}

	testRole := CreateTestRole(testNamespace, "existing-role", AppLabels)
	_, err = clientset.RbacV1().Roles(testNamespace).Create(context.TODO(), testRole, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Role", err)
	}

	rb3 := CreateTestRoleBinding(
		testNamespace,
		"test-rb3",
		"non-existing-service-account",
		&rbacv1.RoleRef{
			Kind: "Role",
			Name: "existing-role",
		})
	_, err = clientset.RbacV1().RoleBindings(testNamespace).Create(context.TODO(), rb3, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "RoleBinding: rb3", err)
	}

	rb4 := CreateTestRoleBinding(
		testNamespace,
		"test-rb4",
		"non-existing-service-account",
		&rbacv1.RoleRef{
			Kind: "Role",
			Name: "existing-role",
		})

	sa4 := CreateTestServiceAccount(testNamespace, "existing-service-account", AppLabels)
	_, err = clientset.CoreV1().ServiceAccounts(testNamespace).Create(context.TODO(), sa4, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "ServiceAccount", err)
	}

	rbacSubject := CreateTestRbacSubject(testNamespace, "existing-service-account")
	rb4.Subjects = append(rb4.Subjects, *rbacSubject)
	_, err = clientset.RbacV1().RoleBindings(testNamespace).Create(context.TODO(), rb4, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "RoleBinding: rb4", err)
	}
	return clientset
}

func TestProcessNamespaceRoleBindings(t *testing.T) {
	clientset := createTestRoleBindings(t)

	unusedRoleBindings, err := processNamespaceRoleBindings(clientset, testNamespace, &filters.Options{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	expectedRoleBindingNames := []string{"test-rb1", "test-rb2", "test-rb3"}

	if len(unusedRoleBindings) != len(expectedRoleBindingNames) {
		t.Errorf("Expected %d unused role bindings, got %d", len(expectedRoleBindingNames), len(unusedRoleBindings))
	}

	for i, rb := range unusedRoleBindings {
		if rb.Name != expectedRoleBindingNames[i] {
			t.Errorf("Expected %s, got %s", expectedRoleBindingNames[i], rb.Name)
		}
	}

}

func TestGetUnusedRoleBindingStructured(t *testing.T) {
	clientset := createTestRoleBindings(t)

	opts := common.Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	output, err := GetUnusedRoleBindings(&filters.Options{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedRoleBindingStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"RoleBinding": {
				"test-rb1",
				"test-rb2",
				"test-rb3",
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
