package kor

import (
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

func CreateTestService(namespace, name string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
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
