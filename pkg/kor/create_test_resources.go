package kor

import (
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var testNamespace = "test-namespace"
var AppLabels = map[string]string{}
var UsedLabels = map[string]string{"kor/used": "true"}
var UnusedLabels = map[string]string{"kor/used": "false"}

func CreateTestDeployment(namespace, name string, replicas int32, labels map[string]string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
	}
}

func CreateTestStatefulSet(namespace, name string, replicas int32, labels map[string]string) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
		},
	}
}

func CreateTestService(namespace, name string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
}

func CreateTestPod(namespace, name, serviceAccountName string, volumes []corev1.Volume, labels map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Volumes:            volumes,
			InitContainers:     nil,
			Containers:         nil,
			ServiceAccountName: serviceAccountName,
		},
	}
}

func CreatePersistentVolumeClaimVolumeSource(name string) *corev1.PersistentVolumeClaimVolumeSource {
	return &corev1.PersistentVolumeClaimVolumeSource{
		ClaimName: name,
	}
}

func CreateTestVolume(name, pvcName string) *corev1.Volume {
	pvc := CreatePersistentVolumeClaimVolumeSource(pvcName)
	return &corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: pvc,
		},
	}

}

func CreateTestServiceAccount(namespace, name string, labels map[string]string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    labels,
		},
	}
}

func CreateTestRbacSubject(namespace, serviceAccountName string) *rbacv1.Subject {
	return &rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      serviceAccountName,
		Namespace: namespace,
	}
}

func CreateTestRoleRef(roleName string) *rbacv1.RoleRef {
	return &rbacv1.RoleRef{
		Kind: "Role",
		Name: roleName,
	}
}

func CreateTestRoleBinding(namespace, name, serviceAccountName string, roleRefName *rbacv1.RoleRef) *rbacv1.RoleBinding {
	rbacSubject := CreateTestRbacSubject(namespace, serviceAccountName)
	return &rbacv1.RoleBinding{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Subjects: []rbacv1.Subject{
			*rbacSubject,
		},
		RoleRef: *roleRefName,
	}
}

func CreateTestClusterRoleBinding(namespace, name, serviceAccountName string) *rbacv1.ClusterRoleBinding {
	rbacSubject := CreateTestRbacSubject(namespace, serviceAccountName)
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
		Subjects: []rbacv1.Subject{
			*rbacSubject,
		},
	}
}

func createPolicyRule() *rbacv1.PolicyRule {
	return &rbacv1.PolicyRule{
		Verbs:     []string{"get"},
		Resources: []string{"pods"},
	}
}

func CreateTestRole(namespace, name string, labels map[string]string) *rbacv1.Role {
	policyRule := createPolicyRule()
	return &rbacv1.Role{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Rules: []rbacv1.PolicyRule{*policyRule},
	}
}

func CreateTestEndpoint(namespace, name string, endpointSubsetCount int, labels map[string]string) *corev1.Endpoints {
	return &corev1.Endpoints{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    labels,
		},
		Subsets: make([]corev1.EndpointSubset, endpointSubsetCount),
	}
}
func CreateTestHpa(namespace, name, deploymentName string, minReplicas, maxReplicas int32, labels map[string]string) *autoscalingv2.HorizontalPodAutoscaler {
	return &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			MinReplicas: &minReplicas,
			MaxReplicas: maxReplicas,
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Kind: "Deployment",
				Name: deploymentName,
			},
		},
	}
}

func CreateTestIngress(namespace, name, ServiceName, secretName string, labels map[string]string) *networkingv1.Ingress {
	ingressRule := networkingv1.IngressRule{
		Host: "test.com",
		IngressRuleValue: networkingv1.IngressRuleValue{
			HTTP: &networkingv1.HTTPIngressRuleValue{
				Paths: []networkingv1.HTTPIngressPath{
					{
						Path: "/path",
						Backend: networkingv1.IngressBackend{
							Service: &networkingv1.IngressServiceBackend{
								Name: ServiceName,
							},
						},
					},
				},
			},
		},
	}
	ingressTls := networkingv1.IngressTLS{
		SecretName: secretName,
	}

	return &networkingv1.Ingress{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{ingressRule},
			TLS:   []networkingv1.IngressTLS{ingressTls},
		},
	}
}

func CreateTestPvc(namespace, name string, labels map[string]string, storageClass string) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: &storageClass,
		},
	}
}

func CreateTestPv(name, phase string, labels map[string]string, storageClass string) *corev1.PersistentVolume {
	return &corev1.PersistentVolume{
		ObjectMeta: v1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: corev1.PersistentVolumeSpec{
			StorageClassName: storageClass,
		},
		Status: corev1.PersistentVolumeStatus{
			Phase: corev1.PersistentVolumePhase(phase),
		},
	}
}

func CreateTestStorageClass(name, provisioner string) *storagev1.StorageClass {
	return &storagev1.StorageClass{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
		Provisioner: provisioner,
	}
}

func CreateTestPdb(namespace, name string, matchLabels, pdbLabels map[string]string) *policyv1.PodDisruptionBudget {
	return &policyv1.PodDisruptionBudget{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    pdbLabels,
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MinAvailable: nil,
			Selector: &v1.LabelSelector{
				MatchLabels: matchLabels,
			},
		},
	}
}

func CreateTestSecret(namespace, name string, labels map[string]string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    labels,
		},
	}
}

func CreateTestConfigmap(namespace, name string, labels map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    labels,
		},
	}
}

func CreateTestJob(namespace, name string, status *batchv1.JobStatus, labels map[string]string) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test",
							Image: "test",
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
		Status: *status,
	}
}

func CreateTestReplicaSet(namespace, name string, specReplicas *int32, status *appsv1.ReplicaSetStatus) *appsv1.ReplicaSet {
	return &appsv1.ReplicaSet{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: specReplicas,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test",
							Image: "test",
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
		Status: *status,
	}
}

func CreateTestClusterRole(name string, labels map[string]string, matchLabels ...v1.LabelSelector) *rbacv1.ClusterRole {
	policyRule := createPolicyRule()
	return &rbacv1.ClusterRole{
		ObjectMeta: v1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		AggregationRule: &rbacv1.AggregationRule{
			ClusterRoleSelectors: matchLabels,
		},
		Rules: []rbacv1.PolicyRule{*policyRule},
	}
}

func CreateTestClusterRoleBindingRoleRef(namespace, name, serviceAccountName string, roleRefName *rbacv1.RoleRef) *rbacv1.ClusterRoleBinding {
	rbacSubject := CreateTestRbacSubject(namespace, serviceAccountName)
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
		Subjects: []rbacv1.Subject{
			*rbacSubject,
		},
		RoleRef: *roleRefName,
	}
}

func CreateTestRoleRefForClusterRole(roleName string) *rbacv1.RoleRef {
	return &rbacv1.RoleRef{
		Kind: "ClusterRole",
		Name: roleName,
	}
}

func CreateTestDaemonSet(namespace, name string, labels map[string]string, status *appsv1.DaemonSetStatus) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    labels,
		},
		Status: *status,
	}
}

func CreateTestUnstructered(kind, apiVersion, namespace, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       kind,
			"apiVersion": apiVersion,
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{},
		},
	}
}

func CreateTestNetworkPolicy(name, namespace string, podSelector v1.LabelSelector, labels map[string]string) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: podSelector,
		},
	}
}
