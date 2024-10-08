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

func TestGetUnusedArgoRolloutsStructured(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientsetargorollouts := createClientSetTestArgoRollouts(t)

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
	deployment1 := CreateTestDeployment(testNamespace, deploymentName, 0, AppLabels)
	_, err = clientset.AppsV1().Deployments(testNamespace).Create(context.TODO(), deployment1, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating fake deployment: %v", err)
	}

	rollout1 := CreateTestArgoRolloutWithDeployment(testNamespace, "test-rollout", deployment1, AppLabels)
	_, err = clientsetargorollouts.ArgoprojV1alpha1().Rollouts(testNamespace).Create(context.TODO(), rollout1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake argo rollout: %v", err)
	}

	err = clientset.AppsV1().Deployments(testNamespace).Delete(context.TODO(), deploymentName, v1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Error creating fake argo rollout: %v", err)
	}

	output, err := GetUnusedArgoRollouts(&filters.Options{}, clientset, clientsetargorollouts, "json", opts)

	if err != nil {
		t.Fatalf("Error calling GetUnusedArgoRolloutsStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"ArgoRollout": {
				"test-rollout",
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
