package kor

import (
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

var testNamespace = "test-namespace"

func CreateTestDeployment(namespace, name string, replicas int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
	}
}

func CreateTestStatefulSet(namespace, name string, replicas int32) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
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

func CreateTestVolume(name string) *corev1.Volume {
	return &corev1.Volume{
		Name:         name,
		VolumeSource: corev1.VolumeSource{},
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

func CreateTestRoleBinding(namespace, name, serviceAccountName string) *rbacv1.RoleBinding {
	rbacSubject := CreateTestRbacSubject(namespace, serviceAccountName)
	return &rbacv1.RoleBinding{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Subjects: []rbacv1.Subject{
			*rbacSubject,
		},
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

func CreateTestIngress(namespace, name, ServiceName string) *networkingv1.Ingress {
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

	return &networkingv1.Ingress{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{ingressRule},
		},
	}
}
