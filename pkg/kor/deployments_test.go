package kor

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	fakeargorollouts "github.com/argoproj/argo-rollouts/pkg/client/clientset/versioned/fake"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

func createTestDeployments(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	deployment1 := CreateTestDeployment(testNamespace, "test-deployment1", 0, AppLabels)
	_, err = clientset.AppsV1().Deployments(testNamespace).Create(context.TODO(), deployment1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake deployment: %v", err)
	}

	deployment2 := CreateTestDeployment(testNamespace, "test-deployment2", 1, AppLabels)
	_, err = clientset.AppsV1().Deployments(testNamespace).Create(context.TODO(), deployment2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake deployment: %v", err)
	}

	deployment3 := CreateTestDeployment(testNamespace, "test-deployment3", 0, UsedLabels)
	_, err = clientset.AppsV1().Deployments(testNamespace).Create(context.TODO(), deployment3, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake deployment: %v", err)
	}

	deployment4 := CreateTestDeployment(testNamespace, "test-deployment4", 1, UnusedLabels)
	_, err = clientset.AppsV1().Deployments(testNamespace).Create(context.TODO(), deployment4, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake deployment: %v", err)
	}

	return clientset
}

func createClientSetTestArgoRollouts(t *testing.T) *fakeargorollouts.Clientset {
	return fakeargorollouts.NewSimpleClientset()
}

func TestProcessNamespaceDeployments(t *testing.T) {
	clientset := createTestDeployments(t)
	clientsetargorollouts := createClientSetTestArgoRollouts(t)

	deploymentsWithoutReplicas, err := processNamespaceDeployments(clientset, clientsetargorollouts, testNamespace, &filters.Options{})
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(deploymentsWithoutReplicas) != 2 {
		t.Errorf("Expected 1 deployment without replicas, got %d", len(deploymentsWithoutReplicas))
	}

	if deploymentsWithoutReplicas[0].Name != "test-deployment1" && deploymentsWithoutReplicas[1].Name != "test-deployment4" {
		t.Errorf("Expected 'test-deployment1', 'test-deployment4',got %s, %s", deploymentsWithoutReplicas[0], deploymentsWithoutReplicas[1])
	}
}

func TestGetUnusedDeploymentsStructured(t *testing.T) {
	clientset := createTestDeployments(t)
	clientsetargorollouts := createClientSetTestArgoRollouts(t)

	opts := common.Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	output, err := GetUnusedDeployments(&filters.Options{}, clientset, clientsetargorollouts, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedDeploymentsStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"Deployment": {
				"test-deployment1",
				"test-deployment4",
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

func TestGetUnusedDeploymentsWithArgoRolloutStructured(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	opts := common.Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}
	deploymentName := "test-deployment1"
	deplomentWorkLoadRefNoReplicas := CreateTestDeployment(testNamespace, deploymentName, 0, AppLabels)
	_, err = clientset.AppsV1().Deployments(testNamespace).Create(context.TODO(), deplomentWorkLoadRefNoReplicas, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake deployment: %v", err)
	}

	clientsetargorollouts := createClientSetTestArgoRollouts(t)
	CreateTestArgoRolloutWithDeployment(testNamespace, deploymentName, deplomentWorkLoadRefNoReplicas, AppLabels)

	output, err := GetUnusedDeployments(&filters.Options{}, clientset, clientsetargorollouts, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedDeploymentsStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"Deployment": {
				deploymentName,
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
