package kor

import (
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

var testNamespace = "test-namespace"

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

func CreateTestPod(namespace, name, serviceAccountName string, volumes []corev1.Volume) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
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

func CreateTestServiceAccount(namespace, name string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
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

func CreateTestRole(namespace, name string) *rbacv1.Role {
	policyRule := createPolicyRule()
	return &rbacv1.Role{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Rules: []rbacv1.PolicyRule{*policyRule},
	}
}

func CreateTestEndpoint(namespace, name string, endpointSubsetCount int) *corev1.Endpoints {
	return &corev1.Endpoints{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Subsets: make([]corev1.EndpointSubset, endpointSubsetCount),
	}
}
func CreateTestHpa(namespace, name, deploymentName string, minReplicas, maxReplicas int32) *autoscalingv2.HorizontalPodAutoscaler {
	return &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
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

func CreateTestIngress(namespace, name, ServiceName, secretName string) *networkingv1.Ingress {
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
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{ingressRule},
			TLS:   []networkingv1.IngressTLS{ingressTls},
		},
	}
}

func CreateTestPvc(namespace, name string) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
}

func CreateTestPv(name, phase string) *corev1.PersistentVolume {
	return &corev1.PersistentVolume{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
		Status: corev1.PersistentVolumeStatus{
			Phase: corev1.PersistentVolumePhase(phase),
		},
	}
}

func CreateTestPdb(namespace, name string, matchLabels map[string]string) *policyv1.PodDisruptionBudget {
	return &policyv1.PodDisruptionBudget{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MinAvailable: nil,
			Selector: &v1.LabelSelector{
				MatchLabels: matchLabels,
			},
		},
	}
}

func CreateTestSecret(namespace, name string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
}

func CreateTestConfigmap(namespace, name string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
}

func CreateTestJob(namespace, name string, status *batchv1.JobStatus) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
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
