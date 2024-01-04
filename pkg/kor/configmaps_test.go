package kor

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
)

func createTestConfigmaps(clientset *fake.Clientset, t *testing.T) {

	configmap1 := CreateTestConfigmap(testNamespace, "configmap-1")
	configmap2 := CreateTestConfigmap(testNamespace, "configmap-2")
	configmap3 := CreateTestConfigmap(testNamespace, "configmap-3")

	pod1 := CreateTestPod(testNamespace, "cm-pod-1", "", []corev1.Volume{
		{Name: "vol-1", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: configmap1.ObjectMeta.Name}}}},
	})

	pod2 := CreateTestPod(testNamespace, "cm-pod-2", "", nil)
	pod2.Spec.Containers = []corev1.Container{
		{
			Env: []corev1.EnvVar{
				{Name: "ENV_VAR_1", ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: configmap1.ObjectMeta.Name}}}},
			},
		},
	}

	pod3 := CreateTestPod(testNamespace, "cm-pod-3", "", nil)
	pod3.Spec.Containers = []corev1.Container{
		{
			EnvFrom: []corev1.EnvFromSource{
				{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: configmap2.ObjectMeta.Name}}},
			},
		},
	}

	pod4 := CreateTestPod(testNamespace, "cm-pod-4", "", nil)
	pod4.Spec.InitContainers = []corev1.Container{
		{
			Env: []corev1.EnvVar{
				{Name: "INIT_ENV_VAR_1", ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: configmap2.ObjectMeta.Name}}}},
			},
		},
	}

	_, err := clientset.CoreV1().ConfigMaps(testNamespace).Create(context.TODO(), configmap1, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake configmap: %v", err)
	}

	_, err = clientset.CoreV1().ConfigMaps(testNamespace).Create(context.TODO(), configmap2, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake configmap: %v", err)
	}

	_, err = clientset.CoreV1().ConfigMaps(testNamespace).Create(context.TODO(), configmap3, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake configmap: %v", err)
	}

	_, err = clientset.CoreV1().Pods(testNamespace).Create(context.TODO(), pod1, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake pod: %v", err)
	}

	_, err = clientset.CoreV1().Pods(testNamespace).Create(context.TODO(), pod2, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake pod: %v", err)
	}

	_, err = clientset.CoreV1().Pods(testNamespace).Create(context.TODO(), pod3, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake pod: %v", err)
	}

	_, err = clientset.CoreV1().Pods(testNamespace).Create(context.TODO(), pod4, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake pod: %v", err)
	}

}

func createTestConfigmapsClient(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
	}, metav1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	createTestConfigmaps(clientset, t)

	return clientset
}

func TestRetrieveConfigMapNames(t *testing.T) {
	clientset := createTestConfigmapsClient(t)

	configMapNames, err := retrieveConfigMapNames(clientset, testNamespace, &FilterOptions{})

	if err != nil {
		t.Fatalf("Error retrieving configmap names: %v", err)
	}

	expectedConfigMapNames := []string{"configmap-1", "configmap-2", "configmap-3"}
	if !equalSlices(configMapNames, expectedConfigMapNames) {
		t.Errorf("Expected configmap names %v, got %v", expectedConfigMapNames, configMapNames)
	}
}

func TestProcessNamespaceCM(t *testing.T) {
	clientset := createTestConfigmapsClient(t)

	diff, err := processNamespaceCM(clientset, testNamespace, &FilterOptions{})
	if err != nil {
		t.Fatalf("Error processing namespace CM: %v", err)
	}

	unusedConfigmaps := []string{"configmap-3"}
	if !equalSlices(diff, unusedConfigmaps) {
		t.Errorf("Expected diff %v, got %v", unusedConfigmaps, diff)
	}
}

func TestRetrieveUsedCM(t *testing.T) {
	clientset := createTestConfigmapsClient(t)

	volumesCM, envCM, envFromCM, envFromContainerCM, envFromInitContainerCM, err := retrieveUsedCM(clientset, testNamespace)

	if err != nil {
		t.Fatalf("Error retrieving used ConfigMaps: %v", err)
	}

	expectedVolumesCM := []string{"configmap-1", "kube-root-ca.crt"}
	if !equalSlices(volumesCM, expectedVolumesCM) {
		t.Errorf("Expected volume configmaps %v, got %v", expectedVolumesCM, volumesCM)
	}

	expectedEnvCM := []string{"configmap-1"}
	if !equalSlices(envCM, expectedEnvCM) {
		t.Errorf("Expected env configmaps %v, got %v", expectedEnvCM, envCM)
	}

	expectedEnvFromCM := []string{"configmap-2"}
	if !equalSlices(envFromCM, expectedEnvFromCM) {
		t.Errorf("Expected envFrom configmaps %v, got %v", expectedEnvFromCM, envFromCM)
	}

	expectedEnvFromContainerCM := []string{"configmap-2"}
	if !equalSlices(envFromContainerCM, expectedEnvFromContainerCM) {
		t.Errorf("Expected envFrom configmaps %v, got %v", expectedEnvFromContainerCM, envFromContainerCM)
	}

	expectedEnvFromInitContainerCM := []string{"configmap-2"}
	if !equalSlices(envFromInitContainerCM, expectedEnvFromInitContainerCM) {
		t.Errorf("Expected initContainer env configmaps %v, got %v", expectedEnvFromInitContainerCM, envFromInitContainerCM)
	}

}

func TestGetUnusedConfigmapsStructured(t *testing.T) {
	clientset := createTestConfigmapsClient(t)

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

	output, err := GetUnusedConfigmaps(includeExcludeLists, &FilterOptions{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedConfigmapsStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"ConfigMap": {"configmap-3"},
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
