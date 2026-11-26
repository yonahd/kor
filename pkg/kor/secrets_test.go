package kor

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	clienttesting "k8s.io/client-go/testing"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

func createTestSecrets(t *testing.T) *fake.Clientset {
	clientset := fake.NewClientset()

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
	secret6 := CreateTestSecret(testNamespace, "test-secret6", AppLabels)

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
					ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: secret1.Name}}},
				},
			},
		},
	}

	pod4 := CreateTestPod(testNamespace, "pod-4", "", nil, AppLabels)
	pod4.Spec.Containers = []corev1.Container{
		{
			EnvFrom: []corev1.EnvFromSource{
				{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: secret1.Name}}},
			},
		},
	}

	pod5 := CreateTestPod(testNamespace, "pod-5", "", nil, AppLabels)
	pod5.Spec.InitContainers = []corev1.Container{
		{
			Env: []corev1.EnvVar{
				{
					Name:      "INIT_ENV_VAR_1",
					ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: secret1.Name}}},
				},
			},
		},
	}

	pod6 := CreateTestPod(testNamespace, "pod-6", "", nil, AppLabels)
	pod6.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
		{Name: secret1.Name},
		{Name: secret2.Name},
	}

	pod7 := CreateTestPod(testNamespace, "pod-7", "", nil, AppLabels)
	pod7.Spec.InitContainers = []corev1.Container{
		{
			EnvFrom: []corev1.EnvFromSource{
				{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: secret6.Name}}},
			},
		},
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

	_, err = clientset.CoreV1().Pods(testNamespace).Create(context.TODO(), pod7, v1.CreateOptions{})
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

	_, err = clientset.CoreV1().Secrets(testNamespace).Create(context.TODO(), secret6, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake %s: %v", "Secret", err)
	}

	return clientset
}

func TestRetrieveIngressTLS(t *testing.T) {
	clientset := fake.NewClientset()

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

	expectedInitContainerEnvSecrets := []string{"test-secret1", "test-secret6"}
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
	clientset := fake.NewClientset()

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

	unusedSecrets, err := processNamespaceSecret(clientset, testNamespace, nil, &filters.Options{}, common.Opts{})
	if err != nil {
		t.Fatalf("Error retrieving unused secrets: %v", err)
	}

	if len(unusedSecrets) != 2 {
		t.Errorf("Expected 2 unused Secret objects, got %d", len(unusedSecrets))
	}

	if !resourceInfoContains(unusedSecrets, "test-secret3") {
		t.Error("Expected specific Secret in the list")
	}

}

func TestProcessNamespaceSecretCSINodePublishSecretRef(t *testing.T) {
	clientset := fake.NewClientset()

	usedSecret := CreateTestSecret(testNamespace, "csi-credentials", AppLabels)
	unusedSecret := CreateTestSecret(testNamespace, "orphan", AppLabels)

	pod := CreateTestPod(testNamespace, "csi-pod", "", []corev1.Volume{
		{
			Name: "secrets-store",
			VolumeSource: corev1.VolumeSource{CSI: &corev1.CSIVolumeSource{
				Driver:               "secrets-store.csi.k8s.io",
				NodePublishSecretRef: &corev1.LocalObjectReference{Name: usedSecret.Name},
			}},
		},
	}, AppLabels)

	if _, err := clientset.CoreV1().Pods(testNamespace).Create(context.TODO(), pod, v1.CreateOptions{}); err != nil {
		t.Fatalf("Error creating fake pod: %v", err)
	}
	for _, secret := range []*corev1.Secret{usedSecret, unusedSecret} {
		if _, err := clientset.CoreV1().Secrets(testNamespace).Create(context.TODO(), secret, v1.CreateOptions{}); err != nil {
			t.Fatalf("Error creating fake secret: %v", err)
		}
	}

	unusedSecrets, err := processNamespaceSecret(clientset, testNamespace, nil, &filters.Options{}, common.Opts{})
	if err != nil {
		t.Fatalf("Error retrieving unused secrets: %v", err)
	}

	if resourceInfoContains(unusedSecrets, usedSecret.Name) {
		t.Errorf("Secret %s is referenced by a CSI volume nodePublishSecretRef and must not be reported as unused", usedSecret.Name)
	}
	if !resourceInfoContains(unusedSecrets, unusedSecret.Name) {
		t.Errorf("Expected %s to be reported as unused", unusedSecret.Name)
	}
}

func TestProcessNamespaceSecretPersistentVolumeCSIRefs(t *testing.T) {
	clientset := fake.NewClientset()

	stageSecret := CreateTestSecret(testNamespace, "smb-credentials", AppLabels)
	expandSecret := CreateTestSecret(testNamespace, "expand-credentials", AppLabels)
	unusedSecret := CreateTestSecret(testNamespace, "orphan", AppLabels)
	otherNamespaceSecret := CreateTestSecret("other-namespace", "smb-credentials", AppLabels)

	pv := &corev1.PersistentVolume{
		ObjectMeta: v1.ObjectMeta{Name: "smb-static"},
		Spec: corev1.PersistentVolumeSpec{
			PersistentVolumeSource: corev1.PersistentVolumeSource{CSI: &corev1.CSIPersistentVolumeSource{
				Driver:                    "smb.csi.k8s.io",
				VolumeHandle:              "smb-server/share",
				NodeStageSecretRef:        &corev1.SecretReference{Name: stageSecret.Name, Namespace: testNamespace},
				ControllerExpandSecretRef: &corev1.SecretReference{Name: expandSecret.Name, Namespace: testNamespace},
			}},
		},
	}
	if _, err := clientset.CoreV1().PersistentVolumes().Create(context.TODO(), pv, v1.CreateOptions{}); err != nil {
		t.Fatalf("Error creating fake persistent volume: %v", err)
	}
	for _, secret := range []*corev1.Secret{stageSecret, expandSecret, unusedSecret, otherNamespaceSecret} {
		if _, err := clientset.CoreV1().Secrets(secret.Namespace).Create(context.TODO(), secret, v1.CreateOptions{}); err != nil {
			t.Fatalf("Error creating fake secret: %v", err)
		}
	}

	pvSecrets, err := retrievePersistentVolumeSecrets(clientset)
	if err != nil {
		t.Fatalf("Error retrieving PersistentVolume secrets: %v", err)
	}

	unusedSecrets, err := processNamespaceSecret(clientset, testNamespace, pvSecrets[testNamespace], &filters.Options{}, common.Opts{})
	if err != nil {
		t.Fatalf("Error retrieving unused secrets: %v", err)
	}
	for _, used := range []string{stageSecret.Name, expandSecret.Name} {
		if resourceInfoContains(unusedSecrets, used) {
			t.Errorf("Secret %s is referenced by a CSI PersistentVolume and must not be reported as unused", used)
		}
	}
	if !resourceInfoContains(unusedSecrets, unusedSecret.Name) {
		t.Errorf("Expected %s to be reported as unused", unusedSecret.Name)
	}

	otherUnused, err := processNamespaceSecret(clientset, "other-namespace", pvSecrets["other-namespace"], &filters.Options{}, common.Opts{})
	if err != nil {
		t.Fatalf("Error retrieving unused secrets: %v", err)
	}
	if !resourceInfoContains(otherUnused, otherNamespaceSecret.Name) {
		t.Errorf("A Secret with the same name in another namespace must still be reported as unused")
	}
}

func countPersistentVolumeLists(clientset *fake.Clientset) int {
	count := 0
	for _, action := range clientset.Actions() {
		if action.Matches("list", "persistentvolumes") {
			count++
		}
	}
	return count
}

func createTestNamespacesWithSecrets(t *testing.T, clientset *fake.Clientset, namespaces ...string) {
	t.Helper()
	for _, ns := range namespaces {
		if _, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{Name: ns},
		}, v1.CreateOptions{}); err != nil {
			t.Fatalf("Error creating namespace %s: %v", ns, err)
		}
		if _, err := clientset.CoreV1().Secrets(ns).Create(context.TODO(), CreateTestSecret(ns, "orphan", AppLabels), v1.CreateOptions{}); err != nil {
			t.Fatalf("Error creating fake secret: %v", err)
		}
	}
}

func TestGetUnusedSecretsListsPersistentVolumesOnce(t *testing.T) {
	clientset := fake.NewClientset()
	createTestNamespacesWithSecrets(t, clientset, "ns-a", "ns-b", "ns-c")

	opts := common.Opts{NoInteractive: true, GroupBy: "namespace"}
	if _, err := GetUnusedSecrets(&filters.Options{}, clientset, "json", opts); err != nil {
		t.Fatalf("Error calling GetUnusedSecrets: %v", err)
	}
	if got := countPersistentVolumeLists(clientset); got != 1 {
		t.Errorf("Expected PersistentVolumes to be listed once for 3 namespaces, got %d", got)
	}

	clientset.ClearActions()
	if _, err := GetUnusedAllNamespaced(&filters.Options{}, clientset, "json", opts); err != nil {
		t.Fatalf("Error calling GetUnusedAllNamespaced: %v", err)
	}
	if got := countPersistentVolumeLists(clientset); got != 1 {
		t.Errorf("Expected GetUnusedAllNamespaced to list PersistentVolumes once for 3 namespaces, got %d", got)
	}
}

func TestGetUnusedSecretsPersistentVolumesForbidden(t *testing.T) {
	clientset := fake.NewClientset()
	createTestNamespacesWithSecrets(t, clientset, testNamespace)
	clientset.PrependReactor("list", "persistentvolumes", func(clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(corev1.Resource("persistentvolumes"), "", errors.New("no access"))
	})

	opts := common.Opts{NoInteractive: true, GroupBy: "namespace"}
	output, err := GetUnusedSecrets(&filters.Options{}, clientset, "json", opts)
	if err != nil {
		t.Fatalf("Expected no error when PersistentVolumes cannot be listed, got: %v", err)
	}
	var actual map[string]map[string][]string
	if err := json.Unmarshal([]byte(output), &actual); err != nil {
		t.Fatalf("Error unmarshaling actual JSON: %v", err)
	}
	if got := actual[testNamespace]["Secret"]; !reflect.DeepEqual(got, []string{"orphan"}) {
		t.Errorf("Expected orphan to be reported as unused, got %v", got)
	}
}

func TestGetUnusedSecretsPersistentVolumesError(t *testing.T) {
	clientset := fake.NewClientset()
	createTestNamespacesWithSecrets(t, clientset, testNamespace)
	clientset.PrependReactor("list", "persistentvolumes", func(clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("connection refused")
	})

	opts := common.Opts{NoInteractive: true, GroupBy: "namespace"}
	if _, err := GetUnusedSecrets(&filters.Options{}, clientset, "json", opts); err == nil {
		t.Fatal("Expected an error when listing PersistentVolumes fails")
	}
}

func TestGetUnusedSecretsEphemeralContainerEnv(t *testing.T) {
	keyRef := CreateTestSecret(testNamespace, "debug-key-ref", AppLabels)
	keyRef.Data = map[string][]byte{"mode": []byte("verbose")}
	envFrom := CreateTestSecret(testNamespace, "debug-env-from", AppLabels)
	envFrom.Data = map[string][]byte{"DEBUG_LEVEL": []byte("1")}
	orphan := CreateTestSecret(testNamespace, "unreferenced", AppLabels)
	pod := CreateTestPod(testNamespace, "debugged-pod", "", nil, AppLabels)
	pod.Spec.Containers = []corev1.Container{{Name: "app", Image: "busybox"}}
	pod.Spec.EphemeralContainers = []corev1.EphemeralContainer{
		{EphemeralContainerCommon: corev1.EphemeralContainerCommon{
			Name:  "debugger",
			Image: "busybox",
			Env: []corev1.EnvVar{{
				Name: "DEBUG_MODE",
				ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: keyRef.Name},
					Key:                  "mode",
				}},
			}},
			EnvFrom: []corev1.EnvFromSource{{
				SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: envFrom.Name}},
			}},
		}},
	}
	clientset := fake.NewClientset(&corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: testNamespace}}, pod, keyRef, envFrom, orphan)

	output, err := GetUnusedSecrets(&filters.Options{}, clientset, "json", common.Opts{GroupBy: "namespace"})
	if err != nil {
		t.Fatalf("Error retrieving unused Secrets: %v", err)
	}
	var got map[string]map[string][]string
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatalf("Error unmarshaling unused Secrets: %v", err)
	}
	if unused := got[testNamespace]["Secret"]; !reflect.DeepEqual(unused, []string{orphan.Name}) {
		t.Errorf("Expected only %s to be unused, got %v", orphan.Name, unused)
	}
}

func TestGetUnusedSecretsStructured(t *testing.T) {
	clientset := createTestSecrets(t)

	opts := common.Opts{
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

func TestFilterOwnerReferencedSecrets(t *testing.T) {
	clientset := fake.NewClientset()

	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: v1.ObjectMeta{Name: testNamespace},
	}, v1.CreateOptions{})

	if err != nil {
		t.Fatalf("Error creating namespace %s: %v", testNamespace, err)
	}

	// Create two secrets - one owned by deployment, one standalone
	// Secret owned by deployment
	ownedSecret := CreateTestSecret(testNamespace, "owned-secret", AppLabels)
	// Add owner reference to deployment
	ownedSecret.OwnerReferences = []v1.OwnerReference{
		{
			Kind: "Deployment",
			Name: "test-deployment",
		},
	}

	// Standalone Secret
	standaloneSecret := CreateTestSecret(testNamespace, "standalone-secret", AppLabels)

	_, err = clientset.CoreV1().Secrets(testNamespace).Create(context.TODO(), ownedSecret, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake secret: %v", err)
	}

	_, err = clientset.CoreV1().Secrets(testNamespace).Create(context.TODO(), standaloneSecret, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake secret: %v", err)
	}

	// Test without filter - should return both
	filterOptsNoSkip := &filters.Options{IgnoreOwnerReferences: false}
	unusedWithoutFilter, err := processNamespaceSecret(clientset, testNamespace, nil, filterOptsNoSkip, common.Opts{})
	if err != nil {
		t.Fatalf("Error retrieving unused secrets: %v", err)
	}

	if len(unusedWithoutFilter) != 2 {
		t.Errorf("Expected 2 unused Secret objects without filter, got %d", len(unusedWithoutFilter))
	}

	// Test with filter - should return only standalone
	filterOptsWithSkip := &filters.Options{IgnoreOwnerReferences: true}
	unusedWithFilter, err := processNamespaceSecret(clientset, testNamespace, nil, filterOptsWithSkip, common.Opts{})
	if err != nil {
		t.Fatalf("Error retrieving unused secrets: %v", err)
	}

	if len(unusedWithFilter) != 1 {
		t.Errorf("Expected 1 unused Secret object with filter, got %d", len(unusedWithFilter))
	}

	if unusedWithFilter[0].Name != "standalone-secret" {
		t.Errorf("Expected standalone-secret to be unused, got %s", unusedWithFilter[0].Name)
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

func TestProcessNamespaceSecretInTreeVolumeSecretRefs(t *testing.T) {
	clientset := fake.NewClientset()

	ref := func(name string) *corev1.LocalObjectReference {
		return &corev1.LocalObjectReference{Name: name}
	}
	volumes := []corev1.Volume{
		{Name: "azure", VolumeSource: corev1.VolumeSource{AzureFile: &corev1.AzureFileVolumeSource{
			SecretName: "azure-storage", ShareName: "share",
		}}},
		{Name: "cephfs", VolumeSource: corev1.VolumeSource{CephFS: &corev1.CephFSVolumeSource{
			Monitors: []string{"10.0.0.1:6789"}, SecretRef: ref("ceph-admin"),
		}}},
		{Name: "rbd", VolumeSource: corev1.VolumeSource{RBD: &corev1.RBDVolumeSource{
			CephMonitors: []string{"10.0.0.1:6789"}, RBDImage: "img", SecretRef: ref("rbd-key"),
		}}},
		{Name: "iscsi", VolumeSource: corev1.VolumeSource{ISCSI: &corev1.ISCSIVolumeSource{
			TargetPortal: "10.0.0.2:3260", IQN: "iqn.2001-04.com.example:storage", SecretRef: ref("chap"),
		}}},
		{Name: "flex", VolumeSource: corev1.VolumeSource{FlexVolume: &corev1.FlexVolumeSource{
			Driver: "example/cifs", SecretRef: ref("cifs-credentials"),
		}}},
	}
	pod := CreateTestPod(testNamespace, "in-tree-pod", "", volumes, AppLabels)
	if _, err := clientset.CoreV1().Pods(testNamespace).Create(context.TODO(), pod, v1.CreateOptions{}); err != nil {
		t.Fatalf("Error creating fake pod: %v", err)
	}

	used := []string{"azure-storage", "ceph-admin", "rbd-key", "chap", "cifs-credentials"}
	for _, name := range append(used, "orphan") {
		secret := CreateTestSecret(testNamespace, name, AppLabels)
		if _, err := clientset.CoreV1().Secrets(testNamespace).Create(context.TODO(), secret, v1.CreateOptions{}); err != nil {
			t.Fatalf("Error creating fake secret: %v", err)
		}
	}

	unusedSecrets, err := processNamespaceSecret(clientset, testNamespace, &filters.Options{}, common.Opts{})
	if err != nil {
		t.Fatalf("Error retrieving unused secrets: %v", err)
	}

	for _, name := range used {
		if resourceInfoContains(unusedSecrets, name) {
			t.Errorf("Secret %s is referenced by an in-tree volume and must not be reported as unused", name)
		}
	}
	if !resourceInfoContains(unusedSecrets, "orphan") {
		t.Errorf("Expected orphan to be reported as unused")
	}
}
