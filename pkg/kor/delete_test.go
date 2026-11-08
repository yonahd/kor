package kor

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	fake "k8s.io/client-go/kubernetes/fake"
)

func TestDeleteResource(t *testing.T) {
	clientset := fake.NewClientset()

	configmap1 := CreateTestConfigmap(testNamespace, "configmap-1", AppLabels)
	_, err := clientset.CoreV1().ConfigMaps(testNamespace).Create(context.TODO(), configmap1, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake configmap: %v", err)
	}
	configmap2 := CreateTestConfigmap(testNamespace, "configmap-2", AppLabels)
	_, err = clientset.CoreV1().ConfigMaps(testNamespace).Create(context.TODO(), configmap2, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake configmap: %v", err)
	}

	tests := []struct {
		name          string
		diff          []ResourceInfo
		resourceType  string
		expectedDiff  []ResourceInfo
		expectedError bool
	}{
		{
			name: "Test deletion confirmation",
			diff: []ResourceInfo{
				{Name: configmap1.Name, Reason: "ConfigMap is not used in any pod or container"},
				{Name: configmap2.Name, Reason: "Marked with unused label"},
			},
			resourceType: "ConfigMap",
			expectedDiff: []ResourceInfo{
				{Name: configmap1.Name + "-DELETED", Reason: "ConfigMap is not used in any pod or container"},
				{Name: configmap2.Name + "-DELETED", Reason: "Marked with unused label"},
			},
			expectedError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deletedDiff, _ := DeleteResource(test.diff, clientset, testNamespace, test.resourceType, true)
			for i, deleted := range deletedDiff {
				if deleted != test.expectedDiff[i] {
					t.Errorf("Expected: %s, Got: %s", test.expectedDiff[i], deleted)
				}
			}
		})
	}
}

func TestDeleteDeleteResourceWithFinalizer(t *testing.T) {
	scheme := runtime.NewScheme()
	gvr := schema.GroupVersionResource{Group: "testgroup", Version: "v1", Resource: "TestResource"}
	testResource := CreateTestUnstructered(gvr.Resource, gvr.GroupVersion().String(), testNamespace, "test-resource")
	testResouceInfo := ResourceInfo{Name: testResource.GetName()}
	dynamicClient := fakedynamic.NewSimpleDynamicClient(scheme, testResource)

	_, err := dynamicClient.Resource(gvr).
		Namespace(testNamespace).
		Create(context.TODO(), testResource, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating test resource: %v", err)
	}

	_, err = dynamicClient.
		Resource(gvr).
		Namespace(testNamespace).
		Patch(context.TODO(), "test-resource", types.MergePatchType,
			[]byte(`{"metadata":{"finalizers":["finalizer1", "finalizer2", "finalizer3"]}}`),
			metav1.PatchOptions{})

	if err != nil {
		t.Fatalf("Error patching test resource: %v", err)
	}

	tests := []struct {
		name          string
		diff          []ResourceInfo
		resourceType  string
		expectedDiff  []string
		expectedError bool
	}{
		{
			name:          "Test deletion confirmation",
			diff:          []ResourceInfo{testResouceInfo},
			expectedDiff:  []string{testResource.GetName() + "-DELETED"},
			expectedError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deletedDiff, _ := DeleteResourceWithFinalizer(test.diff, dynamicClient, testNamespace, gvr, true)

			for i, deleted := range deletedDiff {
				if deleted.Name != test.expectedDiff[i] {
					t.Errorf("Expected: %s, Got: %s", test.expectedDiff[i], deleted)
					resource, err := dynamicClient.Resource(gvr).
						Namespace(testNamespace).
						Get(context.TODO(), deleted.Name, metav1.GetOptions{})
					if err != nil {
						t.Error(err)
					}
					if resource.GetFinalizers() != nil {
						t.Error("Finalizers not patched")
					}
				}
			}

		})
	}
}

func TestFlagDynamicResource(t *testing.T) {
	scheme := runtime.NewScheme()
	gvr := schema.GroupVersionResource{Group: "testgroup", Version: "v1", Resource: "TestResource"}
	testResource := CreateTestUnstructered(gvr.Resource, gvr.GroupVersion().String(), testNamespace, "test-resource")
	testResourceWithLabel := CreateTestUnstructered(gvr.Resource, gvr.GroupVersion().String(), testNamespace, "test-resource-with-label")
	dynamicClient := fakedynamic.NewSimpleDynamicClient(scheme, testResource, testResourceWithLabel)
	testResourceWithLabel.SetLabels(map[string]string{
		"test": "true",
	})

	_, err := dynamicClient.Resource(gvr).
		Namespace(testNamespace).
		Create(context.TODO(), testResource, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating test resource: %v", err)
	}
	_, err = dynamicClient.Resource(gvr).
		Namespace(testNamespace).
		Create(context.TODO(), testResourceWithLabel, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating test resource with finalizers: %v", err)
	}

	tests := []struct {
		name          string
		gvr           schema.GroupVersionResource
		resourceName  string
		labels        bool
		expectedError bool
	}{
		{
			name:          "Test flagging dynamic resource",
			resourceName:  "test-resource",
			labels:        false,
			expectedError: false,
		},
		{
			name:          "Test flagging dynamic resource with labels",
			resourceName:  "test-resource-with-label",
			labels:        true,
			expectedError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := FlagDynamicResource(dynamicClient, testNamespace, gvr, test.resourceName)

			if (err != nil) != test.expectedError {
				t.Errorf("Expected error: %v, Got: %v", test.expectedError, err)
			}
			resource, err := dynamicClient.Resource(gvr).
				Namespace(testNamespace).
				Get(context.TODO(), test.resourceName, metav1.GetOptions{})
			if err != nil {
				t.Error(err)
			}
			if resource.GetLabels()["kor/used"] != "true" {
				t.Errorf("Expected resource flagged as used, Got: %v", resource.GetLabels()["kor/used"])
			}
			if test.labels == true && resource.GetLabels()["test"] != "true" {
				t.Errorf("Resource Lost his labels")
			}
		})
	}
}

func TestDeleteResourceCmd(t *testing.T) {
	deleteMap := DeleteResourceCmd()
	if deleteMap == nil {
		t.Fatal("DeleteResourceCmd returned nil map")
	}

	clientset := fake.NewClientset()

	tests := []struct {
		name   string
		create func(clientset kubernetes.Interface) (string, error)
		exists func(clientset kubernetes.Interface, name string) bool
	}{
		{
			name: "ConfigMap",
			create: func(clientset kubernetes.Interface) (string, error) {
				cm := CreateTestConfigmap(testNamespace, "configmap-1", AppLabels)
				_, err := clientset.CoreV1().ConfigMaps(testNamespace).Create(context.TODO(), cm, metav1.CreateOptions{})
				return cm.Name, err
			},
			exists: func(clientset kubernetes.Interface, name string) bool {
				_, err := clientset.CoreV1().ConfigMaps(testNamespace).Get(context.TODO(), name, metav1.GetOptions{})
				return err == nil
			},
		},
		{
			name: "Secret",
			create: func(clientset kubernetes.Interface) (string, error) {
				secret := CreateTestSecret(testNamespace, "secret-1", AppLabels)
				_, err := clientset.CoreV1().Secrets(testNamespace).Create(context.TODO(), secret, metav1.CreateOptions{})
				return secret.Name, err
			},
			exists: func(clientset kubernetes.Interface, name string) bool {
				_, err := clientset.CoreV1().Secrets(testNamespace).Get(context.TODO(), name, metav1.GetOptions{})
				return err == nil
			},
		},
		{
			name: "Service",
			create: func(clientset kubernetes.Interface) (string, error) {
				service := CreateTestService(testNamespace, "service-1")
				_, err := clientset.CoreV1().Services(testNamespace).Create(context.TODO(), service, metav1.CreateOptions{})
				return service.Name, err
			},
			exists: func(clientset kubernetes.Interface, name string) bool {
				_, err := clientset.CoreV1().Services(testNamespace).Get(context.TODO(), name, metav1.GetOptions{})
				return err == nil
			},
		},
		{
			name: "Deployment",
			create: func(clientset kubernetes.Interface) (string, error) {
				deployment := CreateTestDeployment(testNamespace, "deployment-1", 1, AppLabels)
				_, err := clientset.AppsV1().Deployments(testNamespace).Create(context.TODO(), deployment, metav1.CreateOptions{})
				return deployment.Name, err
			},
			exists: func(clientset kubernetes.Interface, name string) bool {
				_, err := clientset.AppsV1().Deployments(testNamespace).Get(context.TODO(), name, metav1.GetOptions{})
				return err == nil
			},
		},
		{
			name: "HPA",
			create: func(clientset kubernetes.Interface) (string, error) {
				hpa := &autoscalingv1.HorizontalPodAutoscaler{
					ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace, Name: "hpa-1"},
				}
				_, err := clientset.AutoscalingV1().HorizontalPodAutoscalers(testNamespace).Create(context.TODO(), hpa, metav1.CreateOptions{})
				return hpa.Name, err
			},
			exists: func(clientset kubernetes.Interface, name string) bool {
				_, err := clientset.AutoscalingV1().HorizontalPodAutoscalers(testNamespace).Get(context.TODO(), name, metav1.GetOptions{})
				return err == nil
			},
		},
		{
			name: "Ingress",
			create: func(clientset kubernetes.Interface) (string, error) {
				ingress := CreateTestIngress(testNamespace, "ingress-1", "service-1", "", AppLabels)
				_, err := clientset.NetworkingV1().Ingresses(testNamespace).Create(context.TODO(), ingress, metav1.CreateOptions{})
				return ingress.Name, err
			},
			exists: func(clientset kubernetes.Interface, name string) bool {
				_, err := clientset.NetworkingV1().Ingresses(testNamespace).Get(context.TODO(), name, metav1.GetOptions{})
				return err == nil
			},
		},
		{
			name: "PDB",
			create: func(clientset kubernetes.Interface) (string, error) {
				pdb := &policyv1beta1.PodDisruptionBudget{
					ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace, Name: "pdb-1"},
				}
				_, err := clientset.PolicyV1beta1().PodDisruptionBudgets(testNamespace).Create(context.TODO(), pdb, metav1.CreateOptions{})
				return pdb.Name, err
			},
			exists: func(clientset kubernetes.Interface, name string) bool {
				_, err := clientset.PolicyV1beta1().PodDisruptionBudgets(testNamespace).Get(context.TODO(), name, metav1.GetOptions{})
				return err == nil
			},
		},
		{
			name: "Role",
			create: func(clientset kubernetes.Interface) (string, error) {
				role := CreateTestRole(testNamespace, "role-1", AppLabels)
				_, err := clientset.RbacV1().Roles(testNamespace).Create(context.TODO(), role, metav1.CreateOptions{})
				return role.Name, err
			},
			exists: func(clientset kubernetes.Interface, name string) bool {
				_, err := clientset.RbacV1().Roles(testNamespace).Get(context.TODO(), name, metav1.GetOptions{})
				return err == nil
			},
		},
		{
			name: "ClusterRole",
			create: func(clientset kubernetes.Interface) (string, error) {
				role := CreateTestClusterRole("cluster-role-1", AppLabels)
				_, err := clientset.RbacV1().ClusterRoles().Create(context.TODO(), role, metav1.CreateOptions{})
				return role.Name, err
			},
			exists: func(clientset kubernetes.Interface, name string) bool {
				_, err := clientset.RbacV1().ClusterRoles().Get(context.TODO(), name, metav1.GetOptions{})
				return err == nil
			},
		},
		{
			name: "ClusterRoleBinding",
			create: func(clientset kubernetes.Interface) (string, error) {
				binding := CreateTestClusterRoleBinding(testNamespace, "cluster-role-binding-1", "service-account-1")
				_, err := clientset.RbacV1().ClusterRoleBindings().Create(context.TODO(), binding, metav1.CreateOptions{})
				return binding.Name, err
			},
			exists: func(clientset kubernetes.Interface, name string) bool {
				_, err := clientset.RbacV1().ClusterRoleBindings().Get(context.TODO(), name, metav1.GetOptions{})
				return err == nil
			},
		},
		{
			name: "PVC",
			create: func(clientset kubernetes.Interface) (string, error) {
				pvc := CreateTestPvc(testNamespace, "pvc-1", AppLabels, "standard")
				_, err := clientset.CoreV1().PersistentVolumeClaims(testNamespace).Create(context.TODO(), pvc, metav1.CreateOptions{})
				return pvc.Name, err
			},
			exists: func(clientset kubernetes.Interface, name string) bool {
				_, err := clientset.CoreV1().PersistentVolumeClaims(testNamespace).Get(context.TODO(), name, metav1.GetOptions{})
				return err == nil
			},
		},
		{
			name: "StatefulSet",
			create: func(clientset kubernetes.Interface) (string, error) {
				ss := CreateTestStatefulSet(testNamespace, "statefulset-1", 1, AppLabels)
				_, err := clientset.AppsV1().StatefulSets(testNamespace).Create(context.TODO(), ss, metav1.CreateOptions{})
				return ss.Name, err
			},
			exists: func(clientset kubernetes.Interface, name string) bool {
				_, err := clientset.AppsV1().StatefulSets(testNamespace).Get(context.TODO(), name, metav1.GetOptions{})
				return err == nil
			},
		},
		{
			name: "ServiceAccount",
			create: func(clientset kubernetes.Interface) (string, error) {
				sa := CreateTestServiceAccount(testNamespace, "service-account-1", AppLabels)
				_, err := clientset.CoreV1().ServiceAccounts(testNamespace).Create(context.TODO(), sa, metav1.CreateOptions{})
				return sa.Name, err
			},
			exists: func(clientset kubernetes.Interface, name string) bool {
				_, err := clientset.CoreV1().ServiceAccounts(testNamespace).Get(context.TODO(), name, metav1.GetOptions{})
				return err == nil
			},
		},
		{
			name: "PV",
			create: func(clientset kubernetes.Interface) (string, error) {
				pv := CreateTestPv("pv-1", "Bound", AppLabels, "standard")
				_, err := clientset.CoreV1().PersistentVolumes().Create(context.TODO(), pv, metav1.CreateOptions{})
				return pv.Name, err
			},
			exists: func(clientset kubernetes.Interface, name string) bool {
				_, err := clientset.CoreV1().PersistentVolumes().Get(context.TODO(), name, metav1.GetOptions{})
				return err == nil
			},
		},
		{
			name: "Pod",
			create: func(clientset kubernetes.Interface) (string, error) {
				pod := CreateTestPod(testNamespace, "pod-1", "", nil, AppLabels)
				_, err := clientset.CoreV1().Pods(testNamespace).Create(context.TODO(), pod, metav1.CreateOptions{})
				return pod.Name, err
			},
			exists: func(clientset kubernetes.Interface, name string) bool {
				_, err := clientset.CoreV1().Pods(testNamespace).Get(context.TODO(), name, metav1.GetOptions{})
				return err == nil
			},
		},
		{
			name: "Job",
			create: func(clientset kubernetes.Interface) (string, error) {
				job := CreateTestJob(testNamespace, "job-1", &batchv1.JobStatus{}, AppLabels)
				_, err := clientset.BatchV1().Jobs(testNamespace).Create(context.TODO(), job, metav1.CreateOptions{})
				return job.Name, err
			},
			exists: func(clientset kubernetes.Interface, name string) bool {
				_, err := clientset.BatchV1().Jobs(testNamespace).Get(context.TODO(), name, metav1.GetOptions{})
				return err == nil
			},
		},
		{
			name: "ReplicaSet",
			create: func(clientset kubernetes.Interface) (string, error) {
				replicas := int32(1)
				rs := CreateTestReplicaSet(testNamespace, "replicaset-1", &replicas, &appsv1.ReplicaSetStatus{})
				_, err := clientset.AppsV1().ReplicaSets(testNamespace).Create(context.TODO(), rs, metav1.CreateOptions{})
				return rs.Name, err
			},
			exists: func(clientset kubernetes.Interface, name string) bool {
				_, err := clientset.AppsV1().ReplicaSets(testNamespace).Get(context.TODO(), name, metav1.GetOptions{})
				return err == nil
			},
		},
		{
			name: "DaemonSet",
			create: func(clientset kubernetes.Interface) (string, error) {
				ds := CreateTestDaemonSet(testNamespace, "daemonset-1", AppLabels, &appsv1.DaemonSetStatus{})
				_, err := clientset.AppsV1().DaemonSets(testNamespace).Create(context.TODO(), ds, metav1.CreateOptions{})
				return ds.Name, err
			},
			exists: func(clientset kubernetes.Interface, name string) bool {
				_, err := clientset.AppsV1().DaemonSets(testNamespace).Get(context.TODO(), name, metav1.GetOptions{})
				return err == nil
			},
		},
		{
			name: "StorageClass",
			create: func(clientset kubernetes.Interface) (string, error) {
				sc := CreateTestStorageClass("storage-class-1", "kubernetes.io/no-provisioner")
				_, err := clientset.StorageV1().StorageClasses().Create(context.TODO(), sc, metav1.CreateOptions{})
				return sc.Name, err
			},
			exists: func(clientset kubernetes.Interface, name string) bool {
				_, err := clientset.StorageV1().StorageClasses().Get(context.TODO(), name, metav1.GetOptions{})
				return err == nil
			},
		},
		{
			name: "NetworkPolicy",
			create: func(clientset kubernetes.Interface) (string, error) {
				np := CreateTestNetworkPolicy("network-policy-1", testNamespace, AppLabels, metav1.LabelSelector{}, nil, nil)
				_, err := clientset.NetworkingV1().NetworkPolicies(testNamespace).Create(context.TODO(), np, metav1.CreateOptions{})
				return np.Name, err
			},
			exists: func(clientset kubernetes.Interface, name string) bool {
				_, err := clientset.NetworkingV1().NetworkPolicies(testNamespace).Get(context.TODO(), name, metav1.GetOptions{})
				return err == nil
			},
		},
		{
			name: "RoleBinding",
			create: func(clientset kubernetes.Interface) (string, error) {
				rb := CreateTestRoleBinding(testNamespace, "role-binding-1", "service-account-1", CreateTestRoleRef("role-1"))
				_, err := clientset.RbacV1().RoleBindings(testNamespace).Create(context.TODO(), rb, metav1.CreateOptions{})
				return rb.Name, err
			},
			exists: func(clientset kubernetes.Interface, name string) bool {
				_, err := clientset.RbacV1().RoleBindings(testNamespace).Get(context.TODO(), name, metav1.GetOptions{})
				return err == nil
			},
		},
		{
			name: "VolumeAttachment",
			create: func(clientset kubernetes.Interface) (string, error) {
				va := CreateTestVolumeAttachment("volume-attachment-1", "test-attacher", "test-node", "pv-1")
				_, err := clientset.StorageV1().VolumeAttachments().Create(context.TODO(), va, metav1.CreateOptions{})
				return va.Name, err
			},
			exists: func(clientset kubernetes.Interface, name string) bool {
				_, err := clientset.StorageV1().VolumeAttachments().Get(context.TODO(), name, metav1.GetOptions{})
				return err == nil
			},
		},
		{
			name: "PriorityClass",
			create: func(clientset kubernetes.Interface) (string, error) {
				pc := CreateTestPriorityClass("priority-class-1", 100)
				_, err := clientset.SchedulingV1().PriorityClasses().Create(context.TODO(), pc, metav1.CreateOptions{})
				return pc.Name, err
			},
			exists: func(clientset kubernetes.Interface, name string) bool {
				_, err := clientset.SchedulingV1().PriorityClasses().Get(context.TODO(), name, metav1.GetOptions{})
				return err == nil
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deleteFunc, ok := deleteMap[test.name]
			if !ok {
				t.Errorf("Expected delete function for %s", test.name)
				return
			}
			name, err := test.create(clientset)
			if err != nil {
				t.Fatalf("Error creating %s: %v", test.name, err)
			}
			if err := deleteFunc(clientset, testNamespace, name); err != nil {
				t.Errorf("Expected no error deleting %s, got %v", test.name, err)
				return
			}
			if test.exists(clientset, name) {
				t.Errorf("Expected %s %q to be deleted", test.name, name)
			}
		})
	}
}

func TestDeleteResourceUnsupportedType(t *testing.T) {
	clientset := fake.NewClientset()

	diff := []ResourceInfo{{Name: "test-resource"}}
	deletedDiff, err := DeleteResource(diff, clientset, testNamespace, "UnsupportedType", true)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(deletedDiff) != 0 {
		t.Errorf("Expected no resources deleted, got %v", deletedDiff)
	}
}

func TestGetResource(t *testing.T) {
	clientset := fake.NewClientset()
	_, err := clientset.CoreV1().ConfigMaps(testNamespace).Create(context.TODO(), CreateTestConfigmap(testNamespace, "configmap-1", AppLabels), metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake configmap: %v", err)
	}

	resource, err := getResource(clientset, testNamespace, "ConfigMap", "configmap-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if _, ok := resource.(*corev1.ConfigMap); !ok {
		t.Errorf("Expected *corev1.ConfigMap, got %T", resource)
	}

	_, err = getResource(clientset, testNamespace, "UnsupportedType", "configmap-1")
	if err == nil {
		t.Error("Expected error for unsupported resource type")
	}

	_, err = getResource(clientset, testNamespace, "ConfigMap", "missing")
	if err == nil {
		t.Error("Expected error for missing resource")
	}
}

func TestFlagResource(t *testing.T) {
	clientset := fake.NewClientset()
	_, err := clientset.CoreV1().ConfigMaps(testNamespace).Create(context.TODO(), CreateTestConfigmap(testNamespace, "configmap-1", AppLabels), metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake configmap: %v", err)
	}

	err = FlagResource(clientset, testNamespace, "ConfigMap", "configmap-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	cm, err := clientset.CoreV1().ConfigMaps(testNamespace).Get(context.TODO(), "configmap-1", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Error getting configmap: %v", err)
	}
	if cm.Labels["kor/used"] != "true" {
		t.Errorf("Expected configmap flagged as used, got labels %v", cm.Labels)
	}

	err = FlagResource(clientset, testNamespace, "UnsupportedType", "configmap-1")
	if err == nil {
		t.Error("Expected error for unsupported resource type")
	}
}

func TestUpdateResource(t *testing.T) {
	clientset := fake.NewClientset()
	_, err := clientset.CoreV1().ConfigMaps(testNamespace).Create(context.TODO(), CreateTestConfigmap(testNamespace, "configmap-1", AppLabels), metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating fake configmap: %v", err)
	}

	resource, err := getResource(clientset, testNamespace, "ConfigMap", "configmap-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	cm := resource.(*corev1.ConfigMap)
	cm.Data = map[string]string{"updated": "true"}

	updated, err := updateResource(clientset, testNamespace, "ConfigMap", cm)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if updatedData := updated.(*corev1.ConfigMap).Data["updated"]; updatedData != "true" {
		t.Errorf("Expected updated data, got %v", updatedData)
	}

	_, err = updateResource(clientset, testNamespace, "UnsupportedType", cm)
	if err == nil {
		t.Error("Expected error for unsupported resource type")
	}
}
