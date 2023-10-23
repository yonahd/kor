package kor

import (
	"errors"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Size holds different types of sizes of Kubernetes resources
type Size struct {
	IntValue       int
	QuantityValues map[v1.ResourceName]resource.Quantity // for future use-cases
}

// GetConfigMapSize returns the total size of the data in a ConfigMap
// This function doesn't take into account the size of the keys or any other metadata about the ConfigMap
func GetConfigMapSize(cm *v1.ConfigMap) int {
	size := 0
	for _, value := range cm.Data {
		size += len(value)
	}
	return size
}

// GetSecretSize returns the total size of the data in a Secret
func GetSecretSize(secret *v1.Secret) int {
	size := 0
	for _, value := range secret.Data {
		size += len(value)
	}
	return size
}

// GetDeploymentSize returns the number of replicas in a Deployment
func GetDeploymentSize(deploy *appsv1.Deployment) int {
	if deploy.Spec.Replicas != nil {
		return int(*deploy.Spec.Replicas)
	}
	return 0 // Default is 0 if Replicas is nil
}

// GetStatefulSetSize returns the number of replicas in a StatefulSet
func GetStatefulSetSize(ss *appsv1.StatefulSet) int {
	if ss.Spec.Replicas != nil {
		return int(*ss.Spec.Replicas)
	}
	return 0 // Default is 0 if Replicas is nil
}

// GetRoleSize returns the number of rules in a Role
func GetRoleSize(role *rbacv1.Role) int {
	return len(role.Rules)
}

// GetHPASize returns the maximum number of replicas that a HPA can scale up to
func GetHPASize(hpa *autoscalingv2.HorizontalPodAutoscaler) int {
	return int(hpa.Spec.MaxReplicas)
}

// GetPVCSize returns the storage capacity requested by a PVC
func GetPVCSize(pvc *v1.PersistentVolumeClaim) resource.Quantity {
	return pvc.Spec.Resources.Requests[v1.ResourceStorage]
}

// GetIngressSize returns the number of rules in an Ingress
func GetIngressSize(ingress *networkingv1.Ingress) int {
	return len(ingress.Spec.Rules)
}

// GetPDBSize returns the minimum number of replicas that must be available during voluntary disruptions
func GetPDBSize(pdb *policyv1.PodDisruptionBudget) int {
	return int(pdb.Spec.MinAvailable.IntVal)
}

// GetPodSize returns the total amount of CPU and memory requested by a Pod
func GetPodSize(pod *v1.Pod) (cpu, memory resource.Quantity) {
	cpu = resource.Quantity{}
	memory = resource.Quantity{}
	for _, container := range pod.Spec.Containers {
		if request, ok := container.Resources.Requests[v1.ResourceCPU]; ok {
			cpu.Add(request)
		}
		if request, ok := container.Resources.Requests[v1.ResourceMemory]; ok {
			memory.Add(request)
		}
	}
	return cpu, memory
}

// GetSize returns the size of a Kubernetes resource.
func GetSize(resource interface{}) (*Size, error) {
	size := &Size{}
	var err error

	switch resource.(type) {
	case *v1.ConfigMap:
		{
			cm := resource.(*v1.ConfigMap)
			size.IntValue = GetConfigMapSize(cm)
		}
	case *v1.Secret:
		{
			secret := resource.(*v1.Secret)
			size.IntValue = GetSecretSize(secret)
		}
	case *appsv1.Deployment:
		{
			deployment := resource.(*appsv1.Deployment)
			size.IntValue = GetDeploymentSize(deployment)
		}
	case *appsv1.StatefulSet:
		{
			statefulSet := resource.(*appsv1.StatefulSet)
			size.IntValue = GetStatefulSetSize(statefulSet)
		}
	case *rbacv1.Role:
		{
			role := resource.(*rbacv1.Role)
			size.IntValue = GetRoleSize(role)
		}
	case *autoscalingv2.HorizontalPodAutoscaler:
		{
			hpa := resource.(*autoscalingv2.HorizontalPodAutoscaler)
			size.IntValue = GetHPASize(hpa)
		}
	case *networkingv1.Ingress:
		{
			ingress := resource.(*networkingv1.Ingress)
			size.IntValue = GetIngressSize(ingress)
		}
	case *policyv1.PodDisruptionBudget:
		{
			pdb := resource.(*policyv1.PodDisruptionBudget)
			size.IntValue = GetPDBSize(pdb)
		}
	case *v1.Pod:
		{
			pod := resource.(*v1.Pod)
			size.QuantityValues[v1.ResourceCPU], size.QuantityValues[v1.ResourceMemory] = GetPodSize(pod)
		}
	case *v1.PersistentVolumeClaim:
		{
			pvc := resource.(*v1.PersistentVolumeClaim)
			size.QuantityValues[v1.ResourceStorage] = GetPVCSize(pvc)
		}
	default:
		return nil, errors.New("not supported resource type")
	}

	return size, err
}
