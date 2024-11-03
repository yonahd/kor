package kor

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/yonahd/kor/pkg/clusterconfig"
	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

func createTestArgoRolloutMultiResources(t *testing.T, rolloutName string, implementationType string) (kubernetes.Interface, clusterconfig.ClientInterface, *appsv1.Deployment) {
	clientsetinterface, _ := NewFakeClientSet(t)
	clientset := clientsetinterface.GetKubernetesClient()
	clientsetargorollouts := clientsetinterface.GetArgoRolloutsClient()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	deploymentName := "test-deployment1"

	deployment := CreateTestDeployment(testNamespace, deploymentName, 0, AppLabels)
	_, err = clientset.AppsV1().Deployments(testNamespace).Create(context.TODO(), deployment, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake deployment: %v", err)
	}

	rollout := CreateTestArgoRolloutWithDeployment(testNamespace, rolloutName, deployment, AppLabels, implementationType)

	_, err = clientsetargorollouts.ArgoprojV1alpha1().Rollouts(testNamespace).Create(context.TODO(), rollout, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake argo rollout: %v", err)
	}

	return clientset, clientsetinterface, deployment

}

func createTestArgoRolloutMultiResourcesWithAnalysis(t *testing.T, rolloutName string, analysisName string, implementationType string) (kubernetes.Interface, clusterconfig.ClientInterface, *appsv1.Deployment) {
	clientsetinterface, _ := NewFakeClientSet(t)
	clientset := clientsetinterface.GetKubernetesClient()
	clientsetargorollouts := clientsetinterface.GetArgoRolloutsClient()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	deploymentName := "test-deployment2"

	deployment := CreateTestDeployment(testNamespace, deploymentName, 0, AppLabels)
	_, err = clientset.AppsV1().Deployments(testNamespace).Create(context.TODO(), deployment, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake deployment: %v", err)
	}
	var rollout *v1alpha1.Rollout
	if implementationType == "bluegreen" {
		rollout = CreateTestArgoRolloutWithDeployment(testNamespace, rolloutName, deployment, AppLabels, implementationType)
	} else {
		rollout = CreateTestArgoRolloutWithDeployment(testNamespace, rolloutName, deployment, AppLabels, implementationType)
	}

	_, err = clientsetargorollouts.ArgoprojV1alpha1().Rollouts(testNamespace).Create(context.TODO(), rollout, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake argo rollout: %v", err)
	}

	analysisTemplate := CreateTestArgoRolloutAnalysis(testNamespace, analysisName, AppLabels)
	_, err = clientsetargorollouts.ArgoprojV1alpha1().AnalysisTemplates(testNamespace).Create(context.TODO(), analysisTemplate, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake analysis template for argo rollout: %v", err)
	}

	return clientset, clientsetinterface, deployment

}

func createTestArgoRolloutMultiResourcesWithClusterAnalysis(t *testing.T, rolloutName string, analysisName string, implementationType string) (kubernetes.Interface, clusterconfig.ClientInterface, *appsv1.Deployment) {
	clientsetinterface, _ := NewFakeClientSet(t)
	clientset := clientsetinterface.GetKubernetesClient()
	clientsetargorollouts := clientsetinterface.GetArgoRolloutsClient()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	deploymentName := "test-deployment2"

	deployment := CreateTestDeployment(testNamespace, deploymentName, 0, AppLabels)
	_, err = clientset.AppsV1().Deployments(testNamespace).Create(context.TODO(), deployment, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake deployment: %v", err)
	}
	var rollout *v1alpha1.Rollout
	if implementationType == "bluegreen" {
		rollout = CreateTestArgoRolloutWithDeployment(testNamespace, rolloutName, deployment, AppLabels, implementationType)
	} else {
		rollout = CreateTestArgoRolloutWithDeployment(testNamespace, rolloutName, deployment, AppLabels, implementationType)
	}

	_, err = clientsetargorollouts.ArgoprojV1alpha1().Rollouts(testNamespace).Create(context.TODO(), rollout, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake argo rollout: %v", err)
	}

	analysisTemplate := CreateTestArgoRolloutClusterAnalysis(analysisName, AppLabels)
	_, err = clientsetargorollouts.ArgoprojV1alpha1().ClusterAnalysisTemplates().Create(context.TODO(), analysisTemplate, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake analysis template for argo rollout: %v", err)
	}

	return clientset, clientsetinterface, deployment

}

func TestGetUnusedArgoRolloutsStructuredByNamespace(t *testing.T) {
	rolloutName := "test-rollout-1"
	implementationType := "canary"
	clientset, clientsetinterface, deployment := createTestArgoRolloutMultiResources(t, rolloutName, implementationType)

	err := clientset.AppsV1().Deployments(testNamespace).Delete(context.TODO(), deployment.Name, v1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Error on delete test deployment %s for argorollout testing: %v", deployment.GetName(), err)
	}

	parseOpts := common.Opts{}
	parseOpts.GroupBy = "namespace"

	opts := &filters.Options{}
	opts.IncludeThirdPartyCrds = append(opts.IncludeThirdPartyCrds, "argo-rollouts")
	opts.IncludeNamespaces = append(opts.IncludeNamespaces, "testNamespace")

	resources := make(map[string]map[string][]ResourceInfo)
	resources[testNamespace] = make(map[string][]ResourceInfo)

	var outputBuffer bytes.Buffer
	var jsonResponse []byte

	GetUnusedCrdsThirdParty("namespace", clientsetinterface, testNamespace, opts, resources, true)
	jsonResponse, err = json.MarshalIndent(resources, "", "  ")
	if err != nil {
		t.Fatalf("Error marshaling jsonResponse: %v", err)
	}

	unused, err := unusedResourceFormatter("json", outputBuffer, parseOpts, jsonResponse)
	if err != nil {
		t.Fatalf("Error on get argorollout unused: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"ArgoRollout": {
				rolloutName,
			},
		},
	}
	var actualOutput map[string]map[string][]string
	if err := json.Unmarshal([]byte(unused), &actualOutput); err != nil {
		t.Fatalf("Error unmarshaling actual output: %v", err)
	}

	if !reflect.DeepEqual(expectedOutput, actualOutput) {
		t.Errorf("Expected output does not match actual output")
	}
}

func TestGetUnusedArgoRolloutsStructuredByResources(t *testing.T) {
	rolloutName := "test-rollout-1"
	implementationType := "canary"
	clientset, clientsetinterface, deployment := createTestArgoRolloutMultiResources(t, rolloutName, implementationType)

	err := clientset.AppsV1().Deployments(testNamespace).Delete(context.TODO(), deployment.Name, v1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Error on delete test deployment %s for argorollout testing: %v", deployment.GetName(), err)
	}

	parseOpts := common.Opts{}
	parseOpts.GroupBy = "resource"

	opts := &filters.Options{}
	opts.IncludeThirdPartyCrds = append(opts.IncludeThirdPartyCrds, "argo-rollouts")
	opts.IncludeNamespaces = append(opts.IncludeNamespaces, "testNamespace")

	resources := make(map[string]map[string][]ResourceInfo)
	resources[testNamespace] = make(map[string][]ResourceInfo)

	var outputBuffer bytes.Buffer
	var jsonResponse []byte

	GetUnusedCrdsThirdParty("resource", clientsetinterface, testNamespace, opts, resources, true)

	jsonResponse, err = json.MarshalIndent(resources, "", "  ")
	if err != nil {
		t.Fatalf("Error marshaling jsonResponse: %v", err)
	}

	unused, err := unusedResourceFormatter("json", outputBuffer, parseOpts, jsonResponse)
	if err != nil {
		t.Fatalf("Error on get argorollout unused: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		"ArgoRollout": {
			testNamespace: {
				rolloutName,
			},
		},
	}

	var actualOutput map[string]map[string][]string
	if err := json.Unmarshal([]byte(unused), &actualOutput); err != nil {
		t.Fatalf("Error unmarshaling actual output: %v", err)
	}

	if !reflect.DeepEqual(expectedOutput, actualOutput) {
		t.Errorf("Expected output does not match actual output")
	}
}

func TestGetUnusedArgoRolloutsCanary(t *testing.T) {
	rolloutName := "test-rollout-2"
	implementationType := "canary"
	clientset, clientsetinterface, deployment := createTestArgoRolloutMultiResources(t, rolloutName, implementationType)

	err := clientset.AppsV1().Deployments(testNamespace).Delete(context.TODO(), deployment.GetName(), v1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Error on delete test deployment %s for argorollout testing: %v", deployment.GetName(), err)
	}

	opts := &filters.Options{}
	opts.IncludeThirdPartyCrds = append(opts.IncludeThirdPartyCrds, "argo-rollouts")
	opts.IncludeNamespaces = append(opts.IncludeNamespaces, testNamespace)

	resources := GetUnusedArgoRollouts(clientsetinterface, testNamespace, opts)
	if err != nil {
		t.Fatalf("Error marshaling jsonResponse: %v", err)
	}
	var argoRolloutsDiffTest []ResourceInfo
	argoRolloutsDiffTest = append(argoRolloutsDiffTest, ResourceInfo{Name: rolloutName, Reason: "Rollout has no deployments"})
	expectedOutput := ResourceDiff{
		"ArgoRollouts",
		argoRolloutsDiffTest,
	}

	if !reflect.DeepEqual(expectedOutput, resources) {
		t.Errorf("Expected output does not match actual output")
	}
}

func TestGetUnusedArgoRolloutsBlueGreen(t *testing.T) {
	rolloutName := "test-rollout-3"
	implementationType := "bluegreen"
	clientset, clientsetinterface, deployment := createTestArgoRolloutMultiResources(t, rolloutName, implementationType)

	err := clientset.AppsV1().Deployments(testNamespace).Delete(context.TODO(), deployment.GetName(), v1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Error on delete test deployment %s for argorollout testing: %v", deployment.GetName(), err)
	}

	opts := &filters.Options{}
	opts.IncludeThirdPartyCrds = append(opts.IncludeThirdPartyCrds, "argo-rollouts")
	opts.IncludeNamespaces = append(opts.IncludeNamespaces, testNamespace)

	resources := GetUnusedArgoRollouts(clientsetinterface, testNamespace, opts)
	if err != nil {
		t.Fatalf("Error marshaling jsonResponse: %v", err)
	}
	var argoRolloutsDiffTest []ResourceInfo
	argoRolloutsDiffTest = append(argoRolloutsDiffTest, ResourceInfo{Name: rolloutName, Reason: "Rollout has no deployments"})
	expectedOutput := ResourceDiff{
		"ArgoRollouts",
		argoRolloutsDiffTest,
	}

	if !reflect.DeepEqual(expectedOutput, resources) {
		t.Errorf("Expected output does not match actual output")
	}
}

func TestGetUnusedArgoRolloutsAnalysisTemplatesCanary(t *testing.T) {
	analysisName := "test-analysys-template-1"
	rolloutName := "test-rollout-4"
	implementationType := "canary"
	clientset, clientsetinterface, deployment := createTestArgoRolloutMultiResourcesWithAnalysis(t, rolloutName, analysisName, implementationType)

	err := clientset.AppsV1().Deployments(testNamespace).Delete(context.TODO(), deployment.GetName(), v1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Error on delete test deployment %s for argorollout testing: %v", deployment.GetName(), err)
	}

	opts := &filters.Options{}
	opts.IncludeThirdPartyCrds = append(opts.IncludeThirdPartyCrds, "argo-rollouts")
	opts.IncludeNamespaces = append(opts.IncludeNamespaces, testNamespace)

	resources := GetUnusedArgoRolloutsAnalysisTemplates(clientsetinterface, testNamespace, opts)
	if err != nil {
		t.Fatalf("Error marshaling jsonResponse: %v", err)
	}

	var argoRolloutsDiffTest []ResourceInfo
	argoRolloutsDiffTest = append(argoRolloutsDiffTest, ResourceInfo{Name: analysisName, Reason: "Argo Rollouts Analysis Templates is not in use"})
	expectedOutput := ResourceDiff{
		"Analysis Templates",
		argoRolloutsDiffTest,
	}

	if !reflect.DeepEqual(expectedOutput, resources) {
		t.Errorf("Expected output does not match actual output")
	}
}

func TestGetUnusedArgoRolloutsAnalysisTemplatesBlueGreen(t *testing.T) {
	analysisName := "test-analysys-template-2"
	rolloutName := "test-rollout-5"
	implementationType := "bluegreen"
	clientset, clientsetinterface, deployment := createTestArgoRolloutMultiResourcesWithAnalysis(t, rolloutName, analysisName, implementationType)

	err := clientset.AppsV1().Deployments(testNamespace).Delete(context.TODO(), deployment.GetName(), v1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Error on delete test deployment %s for argorollout testing: %v", deployment.GetName(), err)
	}

	opts := &filters.Options{}
	opts.IncludeThirdPartyCrds = append(opts.IncludeThirdPartyCrds, "argo-rollouts")
	opts.IncludeNamespaces = append(opts.IncludeNamespaces, testNamespace)

	resources := GetUnusedArgoRolloutsAnalysisTemplates(clientsetinterface, testNamespace, opts)
	if err != nil {
		t.Fatalf("Error marshaling jsonResponse: %v", err)
	}

	var argoRolloutsDiffTest []ResourceInfo
	argoRolloutsDiffTest = append(argoRolloutsDiffTest, ResourceInfo{Name: analysisName, Reason: "Argo Rollouts Analysis Templates is not in use"})
	expectedOutput := ResourceDiff{
		"Analysis Templates",
		argoRolloutsDiffTest,
	}

	if !reflect.DeepEqual(expectedOutput, resources) {
		t.Errorf("Expected output does not match actual output")
	}
}

func TestGetUnusedArgoRolloutsClusterAnalysisTemplatesCanary(t *testing.T) {
	analysisName := "test-analysys-template-3"
	rolloutName := "test-rollout-6"
	implementationType := "canary"
	clientset, clientsetinterface, deployment := createTestArgoRolloutMultiResourcesWithClusterAnalysis(t, rolloutName, analysisName, implementationType)

	err := clientset.AppsV1().Deployments(testNamespace).Delete(context.TODO(), deployment.GetName(), v1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Error on delete test deployment %s for argorollout testing: %v", deployment.GetName(), err)
	}

	opts := &filters.Options{}
	opts.IncludeThirdPartyCrds = append(opts.IncludeThirdPartyCrds, "argo-rollouts")
	opts.IncludeNamespaces = append(opts.IncludeNamespaces, testNamespace)

	resources := GetUnusedArgoRolloutsClusterAnalysisTemplates(clientsetinterface, testNamespace, opts)
	if err != nil {
		t.Fatalf("Error marshaling jsonResponse: %v", err)
	}

	var argoRolloutsDiffTest []ResourceInfo
	argoRolloutsDiffTest = append(argoRolloutsDiffTest, ResourceInfo{Name: analysisName, Reason: "Argo Rollouts Cluster Analysis Templates is not in use"})
	expectedOutput := ResourceDiff{
		"Cluster Analysis Templates",
		argoRolloutsDiffTest,
	}

	if !reflect.DeepEqual(expectedOutput, resources) {
		t.Errorf("Expected output does not match actual output")
	}
}

func TestGetUnusedArgoRolloutsClusterAnalysisTemplatesBlueGreen(t *testing.T) {
	analysisName := "test-analysys-template-4"
	rolloutName := "test-rollout-7"
	implementationType := "bluegreen"
	clientset, clientsetinterface, deployment := createTestArgoRolloutMultiResourcesWithClusterAnalysis(t, rolloutName, analysisName, implementationType)

	err := clientset.AppsV1().Deployments(testNamespace).Delete(context.TODO(), deployment.GetName(), v1.DeleteOptions{})
	if err != nil {
		t.Fatalf("Error on delete test deployment %s for argorollout testing: %v", deployment.GetName(), err)
	}

	opts := &filters.Options{}
	opts.IncludeThirdPartyCrds = append(opts.IncludeThirdPartyCrds, "argo-rollouts")
	opts.IncludeNamespaces = append(opts.IncludeNamespaces, testNamespace)

	resources := GetUnusedArgoRolloutsClusterAnalysisTemplates(clientsetinterface, testNamespace, opts)
	if err != nil {
		t.Fatalf("Error marshaling jsonResponse: %v", err)
	}

	var argoRolloutsDiffTest []ResourceInfo
	argoRolloutsDiffTest = append(argoRolloutsDiffTest, ResourceInfo{Name: analysisName, Reason: "Argo Rollouts Cluster Analysis Templates is not in use"})
	expectedOutput := ResourceDiff{
		"Cluster Analysis Templates",
		argoRolloutsDiffTest,
	}

	if !reflect.DeepEqual(expectedOutput, resources) {
		t.Errorf("Expected output does not match actual output")
	}
}

func init() {
	scheme.Scheme = runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme.Scheme)
}
