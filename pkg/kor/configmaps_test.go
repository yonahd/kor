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

	"github.com/yonahd/kor/pkg/filters"
)

func createTestConfigmaps(t *testing.T) *fake.Clientset {
	clientset := fake.NewSimpleClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
	}, metav1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	configmap1 := CreateTestConfigmap(testNamespace, "configmap-1", AppLabels)
	_, err = clientset.CoreV1().ConfigMaps(testNamespace).Create(context.TODO(), configmap1, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake configmap: %v", err)
	}

	configmap2 := CreateTestConfigmap(testNamespace, "configmap-2", AppLabels)
	_, err = clientset.CoreV1().ConfigMaps(testNamespace).Create(context.TODO(), configmap2, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake configmap: %v", err)
	}

	configmap3 := CreateTestConfigmap(testNamespace, "configmap-3", AppLabels)
	_, err = clientset.CoreV1().ConfigMaps(testNamespace).Create(context.TODO(), configmap3, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake configmap: %v", err)
	}

	configmap4 := CreateTestConfigmap(testNamespace, "configmap-4", UsedLabels)
	_, err = clientset.CoreV1().ConfigMaps(testNamespace).Create(context.TODO(), configmap4, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake configmap: %v", err)
	}

	configmap5 := CreateTestConfigmap(testNamespace, "configmap-5", UnusedLabels)
	_, err = clientset.CoreV1().ConfigMaps(testNamespace).Create(context.TODO(), configmap5, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake configmap: %v", err)
	}

	pod1 := CreateTestPod(testNamespace, "pod-1", "", []corev1.Volume{
		{
			Name:         "vol-1",
			VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: configmap1.ObjectMeta.Name}}},
		},
	}, AppLabels)

	pod2 := CreateTestPod(testNamespace, "pod-2", "", nil, AppLabels)
	pod2.Spec.Containers = []corev1.Container{
		{
			Env: []corev1.EnvVar{
				{
					Name:      "ENV_VAR_1",
					ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: configmap1.ObjectMeta.Name}}},
				},
			},
		},
	}

	pod3 := CreateTestPod(testNamespace, "pod-3", "", nil, AppLabels)
	pod3.Spec.Containers = []corev1.Container{
		{
			EnvFrom: []corev1.EnvFromSource{
				{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: configmap2.ObjectMeta.Name}}},
			},
		},
	}

	pod4 := CreateTestPod(testNamespace, "pod-4", "", nil, AppLabels)
	pod4.Spec.InitContainers = []corev1.Container{
		{
			Env: []corev1.EnvVar{
				{
					Name:      "INIT_ENV_VAR_1",
					ValueFrom: &corev1.EnvVarSource{ConfigMapKeyRef: &corev1.ConfigMapKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: configmap2.ObjectMeta.Name}}},
				},
			},
		},
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

	return clientset
}

func TestRetrieveConfigMapNames(t *testing.T) {
	clientset := createTestConfigmaps(t)

	configMapNames, _, err := retrieveConfigMapNames(clientset, testNamespace, &filters.Options{})

	if err != nil {
		t.Fatalf("Error retrieving configmap names: %v", err)
	}

	expectedConfigMapNames := []string{
		"configmap-1",
		"configmap-2",
		"configmap-3",
	}
	if !equalSlices(configMapNames, expectedConfigMapNames) {
		t.Errorf("Expected configmap names %v, got %v", expectedConfigMapNames, configMapNames)
	}
}

func TestProcessNamespaceCM(t *testing.T) {
	clientset := createTestConfigmaps(t)

	diff, err := processNamespaceCM(clientset, testNamespace, &filters.Options{})
	if err != nil {
		t.Fatalf("Error processing namespace CM: %v", err)
	}

	unusedConfigmaps := []ResourceInfo{
		{Name: "configmap-3", Reason: "ConfigMap is not used in any pod or container"},
		{Name: "configmap-5", Reason: "Marked with unused label"},
	}
	if !equalResourceInfoSlices(diff, unusedConfigmaps) {
		t.Errorf("Expected diff %v, got %v", unusedConfigmaps, diff)
	}
}

func TestRetrieveUsedCM(t *testing.T) {
	clientset := createTestConfigmaps(t)

	volumesCM, envCM, envFromCM, envFromContainerCM, envFromInitContainerCM, err := retrieveUsedCM(clientset, testNamespace)

	if err != nil {
		t.Fatalf("Error retrieving used ConfigMaps: %v", err)
	}

	expectedVolumesCM := []string{
		"configmap-1",
	}
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
	clientset := createTestConfigmaps(t)

	opts := Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	output, err := GetUnusedConfigmaps(&filters.Options{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedConfigmapsStructured: %v", err)
	}

	expectedOutput := map[string]map[string][]string{
		testNamespace: {
			"ConfigMap": {
				"configmap-3",
				"configmap-5",
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
