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

func createTestClusterRoleBindings(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	// Create another namespace for testing cross-namespace service accounts
	_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: "other-namespace"},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", "other-namespace", err)
	}

	// Create test ClusterRoleBinding with non-existing ClusterRole
	crb1 := CreateTestClusterRoleBindingRoleRef(
		testNamespace,
		"test-crb1",
		"sa1",
		&rbacv1.RoleRef{
			Kind: "ClusterRole",
			Name: "non-existing-cluster-role",
		})
	_, err = clientset.RbacV1().ClusterRoleBindings().Create(context.TODO(), crb1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "ClusterRoleBinding: crb1", err)
	}

	// Create test ClusterRoleBinding with non-existing ServiceAccount
	testClusterRole := CreateTestClusterRole("existing-cluster-role", AppLabels)
	_, err = clientset.RbacV1().ClusterRoles().Create(context.TODO(), testClusterRole, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "ClusterRole", err)
	}

	crb2 := CreateTestClusterRoleBindingRoleRef(
		testNamespace,
		"test-crb2",
		"non-existing-service-account",
		&rbacv1.RoleRef{
			Kind: "ClusterRole",
			Name: "existing-cluster-role",
		})
	_, err = clientset.RbacV1().ClusterRoleBindings().Create(context.TODO(), crb2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "ClusterRoleBinding: crb2", err)
	}

	// Create test ClusterRoleBinding that should NOT be unused (has valid SA and CR)
	sa3 := CreateTestServiceAccount(testNamespace, "existing-service-account", AppLabels)
	_, err = clientset.CoreV1().ServiceAccounts(testNamespace).Create(context.TODO(), sa3, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "ServiceAccount", err)
	}

	crb3 := CreateTestClusterRoleBindingRoleRef(
		testNamespace,
		"test-crb3",
		"existing-service-account",
		&rbacv1.RoleRef{
			Kind: "ClusterRole",
			Name: "existing-cluster-role",
		})
	_, err = clientset.RbacV1().ClusterRoleBindings().Create(context.TODO(), crb3, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "ClusterRoleBinding: crb3", err)
	}

	// Create test ClusterRoleBinding with mixed subjects (one exists, one doesn't)
	crb4 := CreateTestClusterRoleBindingRoleRef(
		testNamespace,
		"test-crb4",
		"non-existing-service-account",
		&rbacv1.RoleRef{
			Kind: "ClusterRole",
			Name: "existing-cluster-role",
		})

	// Add existing SA to the subjects
	existingSubject := CreateTestRbacSubject(testNamespace, "existing-service-account")
	crb4.Subjects = append(crb4.Subjects, *existingSubject)
	_, err = clientset.RbacV1().ClusterRoleBindings().Create(context.TODO(), crb4, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "ClusterRoleBinding: crb4", err)
	}

	// Create ClusterRoleBinding with ServiceAccount from different namespace
	sa5 := CreateTestServiceAccount("other-namespace", "cross-ns-service-account", AppLabels)
	_, err = clientset.CoreV1().ServiceAccounts("other-namespace").Create(context.TODO(), sa5, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "ServiceAccount in other-namespace", err)
	}

	crb5 := CreateTestClusterRoleBindingRoleRef(
		"other-namespace",
		"test-crb5",
		"cross-ns-service-account",
		&rbacv1.RoleRef{
			Kind: "ClusterRole",
			Name: "existing-cluster-role",
		})
	_, err = clientset.RbacV1().ClusterRoleBindings().Create(context.TODO(), crb5, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "ClusterRoleBinding: crb5", err)
	}

	return clientset
}

func TestProcessClusterRoleBindings(t *testing.T) {
	clientset := createTestClusterRoleBindings(t)

	unusedClusterRoleBindings, err := processClusterRoleBindings(clientset, &filters.Options{}, common.Opts{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// We expect crb1 (bad ClusterRole) and crb2 (bad SA) to be unused
	// crb3 should be used (valid SA and CR)
	// crb4 should be used (has at least one valid SA)
	// crb5 should be used (valid cross-namespace SA and CR)
	expectedClusterRoleBindingNames := []string{"test-crb1", "test-crb2"}

	if len(unusedClusterRoleBindings) != len(expectedClusterRoleBindingNames) {
		t.Errorf("Expected %d unused cluster role bindings, got %d", len(expectedClusterRoleBindingNames), len(unusedClusterRoleBindings))
	}

	foundNames := make(map[string]bool)
	for _, crb := range unusedClusterRoleBindings {
		foundNames[crb.Name] = true
	}

	for _, expectedName := range expectedClusterRoleBindingNames {
		if !foundNames[expectedName] {
			t.Errorf("Expected to find unused ClusterRoleBinding %s, but it was not in the results", expectedName)
		}
	}

	// Verify reasons
	for _, crb := range unusedClusterRoleBindings {
		if crb.Name == "test-crb1" && crb.Reason != "ClusterRoleBinding references a non-existing ClusterRole" {
			t.Errorf("Expected reason for crb1 to be about non-existing ClusterRole, got: %s", crb.Reason)
		}
		if crb.Name == "test-crb2" && crb.Reason != "ClusterRoleBinding references a non-existing ServiceAccount" {
			t.Errorf("Expected reason for crb2 to be about non-existing ServiceAccount, got: %s", crb.Reason)
		}
	}
}

func TestProcessClusterRoleBindingsWithMixedSubjects(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	// Create a namespace
	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	// Create a valid ClusterRole
	testClusterRole := CreateTestClusterRole("valid-cluster-role", AppLabels)
	_, err = clientset.RbacV1().ClusterRoles().Create(context.TODO(), testClusterRole, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating ClusterRole: %v", err)
	}

	// Create a valid ServiceAccount
	sa := CreateTestServiceAccount(testNamespace, "valid-sa", AppLabels)
	_, err = clientset.CoreV1().ServiceAccounts(testNamespace).Create(context.TODO(), sa, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating ServiceAccount: %v", err)
	}

	// Create ClusterRoleBinding with mixed subjects (User + ServiceAccount)
	crb := CreateTestClusterRoleBindingRoleRef(
		testNamespace,
		"mixed-subjects-crb",
		"valid-sa",
		&rbacv1.RoleRef{
			Kind: "ClusterRole",
			Name: "valid-cluster-role",
		})

	// Add a User subject
	userSubject := rbacv1.Subject{
		Kind:     "User",
		Name:     "alice",
		APIGroup: "rbac.authorization.k8s.io",
	}
	crb.Subjects = append(crb.Subjects, userSubject)

	_, err = clientset.RbacV1().ClusterRoleBindings().Create(context.TODO(), crb, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating ClusterRoleBinding: %v", err)
	}

	unusedClusterRoleBindings, err := processClusterRoleBindings(clientset, &filters.Options{}, common.Opts{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Should be 0 unused since we have mixed subjects (User + ServiceAccount) and we assume Users exist
	if len(unusedClusterRoleBindings) != 0 {
		t.Errorf("Expected 0 unused cluster role bindings with mixed subjects, got %d", len(unusedClusterRoleBindings))
	}
}

func TestGetUnusedClusterRoleBindingStructured(t *testing.T) {
	clientset := createTestClusterRoleBindings(t)

	opts := common.Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	output, err := GetUnusedClusterRoleBindings(&filters.Options{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedClusterRoleBindingStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		"": {
			"ClusterRoleBinding": {
				"test-crb1",
				"test-crb2",
			},
		},
	}

	var actualOutput map[string]map[string][]string
	if err := json.Unmarshal([]byte(output), &actualOutput); err != nil {
		t.Fatalf("Error unmarshaling actual output: %v", err)
	}

	if !reflect.DeepEqual(expectedOutput, actualOutput) {
		t.Errorf("Expected output does not match actual output")
		t.Errorf("Expected: %+v", expectedOutput)
		t.Errorf("Actual: %+v", actualOutput)
	}
}

func TestIsUsingValidServiceAccountClusterScoped(t *testing.T) {
	// Test data
	allServiceAccountNames := map[string]map[string]bool{
		"namespace1": {
			"sa1": true,
			"sa2": true,
		},
		"namespace2": {
			"sa3": true,
		},
	}

	// Test case 1: valid SA exists
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      "sa1",
			Namespace: "namespace1",
		},
	}

	if !isUsingValidServiceAccountClusterScoped(subjects, allServiceAccountNames) {
		t.Errorf("Expected to find valid ServiceAccount sa1 in namespace1")
	}

	// Test case 2: SA doesn't exist
	subjects = []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      "non-existing-sa",
			Namespace: "namespace1",
		},
	}

	if isUsingValidServiceAccountClusterScoped(subjects, allServiceAccountNames) {
		t.Errorf("Expected NOT to find ServiceAccount non-existing-sa")
	}

	// Test case 3: namespace doesn't exist
	subjects = []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      "sa1",
			Namespace: "non-existing-namespace",
		},
	}

	if isUsingValidServiceAccountClusterScoped(subjects, allServiceAccountNames) {
		t.Errorf("Expected NOT to find ServiceAccount sa1 in non-existing-namespace")
	}

	// Test case 4: multiple subjects, at least one valid
	subjects = []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      "non-existing-sa",
			Namespace: "namespace1",
		},
		{
			Kind:      "ServiceAccount",
			Name:      "sa3",
			Namespace: "namespace2",
		},
	}

	if !isUsingValidServiceAccountClusterScoped(subjects, allServiceAccountNames) {
		t.Errorf("Expected to find at least one valid ServiceAccount")
	}
}

func init() {
	scheme.Scheme = runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme.Scheme)
}
