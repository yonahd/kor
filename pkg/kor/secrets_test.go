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

func createTestSecrets(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	secret1 := CreateTestSecret(testNamespace, "test-secret1", AppLabels)
	secret2 := CreateTestSecret(testNamespace, "test-secret2", AppLabels)
	secret3 := CreateTestSecret(testNamespace, "test-secret3", AppLabels)
	secret4 := CreateTestSecret(testNamespace, "test-secret4", UsedLabels)
	secret5 := CreateTestSecret(testNamespace, "test-secret5", UnusedLabels)

	pod1 := CreateTestPod(testNamespace, "pod-1", "", []corev1.Volume{
		{
			Name:         "vol-1",
			VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: "test-secret1"}},
		},
	}, AppLabels)

	pod2 := CreateTestPod(testNamespace, "pod-2", "", []corev1.Volume{
		{
			Name:         "vol-2",
			VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: "test-secret2"}},
		},
	}, AppLabels)

	pod3 := CreateTestPod(testNamespace, "pod-3", "", nil, AppLabels)
	pod3.Spec.Containers = []corev1.Container{
		{
			Env: []corev1.EnvVar{
				{
					Name:      "ENV_VAR_1",
					ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: secret1.ObjectMeta.Name}}},
				},
			},
		},
	}

	pod4 := CreateTestPod(testNamespace, "pod-4", "", nil, AppLabels)
	pod4.Spec.Containers = []corev1.Container{
		{
			EnvFrom: []corev1.EnvFromSource{
				{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: secret1.ObjectMeta.Name}}},
			},
		},
	}

	pod5 := CreateTestPod(testNamespace, "pod-5", "", nil, AppLabels)
	pod5.Spec.InitContainers = []corev1.Container{
		{
			Env: []corev1.EnvVar{
				{
					Name:      "INIT_ENV_VAR_1",
					ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: secret1.ObjectMeta.Name}}},
				},
			},
		},
	}

	pod6 := CreateTestPod(testNamespace, "pod-6", "", nil, AppLabels)
	pod6.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
		{Name: secret1.ObjectMeta.Name},
		{Name: secret2.ObjectMeta.Name},
	}

	_, err = clientset.CoreV1().Pods(testNamespace).Create(context.TODO(), pod1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake pod: %v", err)
	}

	_, err = clientset.CoreV1().Pods(testNamespace).Create(context.TODO(), pod2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake pod: %v", err)
	}

	_, err = clientset.CoreV1().Pods(testNamespace).Create(context.TODO(), pod3, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake pod: %v", err)
	}

	_, err = clientset.CoreV1().Pods(testNamespace).Create(context.TODO(), pod4, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake pod: %v", err)
	}

	_, err = clientset.CoreV1().Pods(testNamespace).Create(context.TODO(), pod5, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake pod: %v", err)
	}

	_, err = clientset.CoreV1().Pods(testNamespace).Create(context.TODO(), pod6, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake pod: %v", err)
	}

	_, err = clientset.CoreV1().Secrets(testNamespace).Create(context.TODO(), secret1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Secret", err)
	}

	_, err = clientset.CoreV1().Secrets(testNamespace).Create(context.TODO(), secret2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Secret", err)
	}

	_, err = clientset.CoreV1().Secrets(testNamespace).Create(context.TODO(), secret3, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Secret", err)
	}

	_, err = clientset.CoreV1().Secrets(testNamespace).Create(context.TODO(), secret4, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Secret", err)
	}

	_, err = clientset.CoreV1().Secrets(testNamespace).Create(context.TODO(), secret5, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Secret", err)
	}

	return clientset
}

func TestRetrieveIngressTLS(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	ingress1 := CreateTestIngress(testNamespace, "test-ingress-1", "my-service-1", "test-secret1", AppLabels)
	appLabels := map[string]string{}
	secret1 := CreateTestSecret(testNamespace, "test-secret1", appLabels)
	secret2 := CreateTestSecret(testNamespace, "test-secret2", appLabels)

	_, err := clientset.NetworkingV1().Ingresses(testNamespace).Create(context.TODO(), ingress1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Ingress", err)
	}

	_, err = clientset.CoreV1().Secrets(testNamespace).Create(context.TODO(), secret1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Secret", err)
	}

	_, err = clientset.CoreV1().Secrets(testNamespace).Create(context.TODO(), secret2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Secret", err)
	}

	tlsSecrets, err := retrieveIngressTLS(clientset, testNamespace)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(tlsSecrets) != 1 {
		t.Errorf("Expected 1 used Secret object, got %d", len(tlsSecrets))
	}

	if tlsSecrets[0] != "test-secret1" {
		t.Errorf("Expected 'test-secret1', got %s", tlsSecrets[0])
	}

}

func TestRetrieveUsedSecret(t *testing.T) {
	clientset := createTestSecrets(t)

	envSecrets, envSecrets2, volumeSecrets, initContainerEnvSecrets, pullSecrets, _, err := retrieveUsedSecret(clientset, testNamespace)
	if err != nil {
		t.Fatalf("Error retrieving used secrets: %v", err)
	}

	expectedVolumeSecrets := []string{
		"test-secret1",
		"test-secret2",
	}
	if !equalSlices(volumeSecrets, expectedVolumeSecrets) {
		t.Errorf("Expected volume secrets %v, got %v", expectedVolumeSecrets, volumeSecrets)
	}

	expectedEnvSecrets := []string{"test-secret1"}
	if !equalSlices(envSecrets, expectedEnvSecrets) {
		t.Errorf("Expected env secrets %v, got %v", expectedEnvSecrets, envSecrets)
	}

	expectedEnvSecrets2 := []string{"test-secret1"}
	if !equalSlices(envSecrets2, expectedEnvSecrets2) {
		t.Errorf("Expected envFrom secrets %v, got %v", expectedEnvSecrets2, envSecrets2)
	}

	expectedInitContainerEnvSecrets := []string{"test-secret1"}
	if !equalSlices(initContainerEnvSecrets, expectedInitContainerEnvSecrets) {
		t.Errorf("Expected initContainer env secrets %v, got %v", expectedInitContainerEnvSecrets, initContainerEnvSecrets)
	}

	expectedPullSecrets := []string{
		"test-secret1",
		"test-secret2",
	}
	if !equalSlices(pullSecrets, expectedPullSecrets) {
		t.Errorf("Expected pull secrets %v, got %v", expectedPullSecrets, pullSecrets)
	}

}

func TestRetrieveSecretNames(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	appLabels := map[string]string{}
	secret1 := CreateTestSecret(testNamespace, "secret-1", appLabels)
	secret2 := CreateTestSecret(testNamespace, "secret-2", appLabels)

	_, err := clientset.CoreV1().Secrets(testNamespace).Create(context.TODO(), secret1, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake secret: %v", err)
	}

	_, err = clientset.CoreV1().Secrets(testNamespace).Create(context.TODO(), secret2, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake secret: %v", err)
	}

	secretNames, _, err := retrieveSecretNames(clientset, testNamespace, &filters.Options{})

	if err != nil {
		t.Fatalf("Error retrieving secret names: %v", err)
	}

	expectedSecretNames := []string{
		"secret-1",
		"secret-2",
	}
	if !equalSlices(secretNames, expectedSecretNames) {
		t.Errorf("Expected secret names %v, got %v", expectedSecretNames, secretNames)
	}
}

func TestProcessNamespaceSecret(t *testing.T) {
	clientset := createTestSecrets(t)

	unusedSecrets, err := processNamespaceSecret(clientset, testNamespace, &filters.Options{})
	if err != nil {
		t.Fatalf("Error retrieving unused secrets: %v", err)
	}

	if len(unusedSecrets) != 2 {
		t.Errorf("Expected 2 used Secret objects, got %d", len(unusedSecrets))
	}

	if !resourceInfoContains(unusedSecrets, "test-secret3") {
		t.Error("Expected specific Secret  in the list")
	}

}

func TestGetUnusedSecretsStructured(t *testing.T) {
	clientset := createTestSecrets(t)

	opts := Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	output, err := GetUnusedSecrets(&filters.Options{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedSecretsStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"Secret": {
				"test-secret3",
				"test-secret5",
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

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func equalResourceInfoSlices(a, b []ResourceInfo) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v.Name != b[i].Name || v.Reason != b[i].Reason {
			return false
		}
	}
	return true
}

func init() {
	scheme.Scheme = runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme.Scheme)
}
