package kor

import (
	"context"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/yonahd/kor/pkg/clusterconfig"
	"github.com/yonahd/kor/pkg/filters"
)

func GetUnusedArgoRollouts(clientsetinterface clusterconfig.ClientInterface, namespace string, filterOpts *filters.Options) ResourceDiff {
	argoRolloutsDiff, err := processNamespaceArgoRollouts(clientsetinterface, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "argorollouts", namespace, err)
	}
	namespaceSADiff := ResourceDiff{
		"ArgoRollouts",
		argoRolloutsDiff,
	}

	return namespaceSADiff
}

func GetUnusedArgoRolloutsAnalysisTemplates(clientsetinterface clusterconfig.ClientInterface, namespace string, filterOpts *filters.Options) ResourceDiff {
	argoRolloutsDiff, err := processNamespaceArgoAnalysisTemplate(clientsetinterface, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "analysis templates from Argo rollouts", namespace, err)
	}

	namespaceSADiff := ResourceDiff{
		"Analysis Templates",
		argoRolloutsDiff,
	}

	return namespaceSADiff
}

func GetUnusedArgoRolloutsClusterAnalysisTemplates(clientsetinterface clusterconfig.ClientInterface, namespace string, filterOpts *filters.Options) ResourceDiff {
	argoRolloutsDiff, err := processNamespaceArgoClusterAnalysisTemplate(clientsetinterface, namespace, filterOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get %s namespace %s: %v\n", "analysis templates from Argo rollouts", namespace, err)
	}

	namespaceSADiff := ResourceDiff{
		"Cluster Analysis Templates",
		argoRolloutsDiff,
	}

	return namespaceSADiff
}

func processNamespaceArgoRollouts(clientsetinterface clusterconfig.ClientInterface, namespace string, filterOpts *filters.Options) ([]ResourceInfo, error) {
	clientset := clientsetinterface.GetKubernetesClient()
	clientsetargorollouts := clientsetinterface.GetArgoRolloutsClient()
	argoRolloutList, err := clientsetargorollouts.ArgoprojV1alpha1().Rollouts(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})

	if err != nil {
		return nil, err
	}

	var argoRolloutWithoutReplicas []ResourceInfo

	for _, argoRollout := range argoRolloutList.Items {
		if pass, _ := filter.SetObject(&argoRollout).Run(filterOpts); pass {
			continue
		}
		if argoRollout.Labels["kor/used"] == "false" {
			reason := "Marked with unused label"
			argoRolloutWithoutReplicas = append(argoRolloutWithoutReplicas, ResourceInfo{Name: argoRollout.Name, Reason: reason})
			continue
		}
		deploymentWorkLoadRef := argoRollout.Spec.WorkloadRef

		if deploymentWorkLoadRef == nil {
			if *argoRollout.Spec.Replicas == 0 {
				reason := "Rollout has 0 replicas"
				argoRolloutWithoutReplicas = append(argoRolloutWithoutReplicas, ResourceInfo{Name: argoRollout.Name, Reason: reason})
			}
		}

		if deploymentWorkLoadRef != nil && deploymentWorkLoadRef.Kind == "Deployment" {
			deploymentItem, _ := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentWorkLoadRef.Name, metav1.GetOptions{})

			if deploymentItem.GetName() == "" {
				reason := "Rollout has no deployments"
				argoRolloutWithoutReplicas = append(argoRolloutWithoutReplicas, ResourceInfo{Name: argoRollout.Name, Reason: reason})
			}
		}
	}

	return argoRolloutWithoutReplicas, nil
}

func processNamespaceArgoAnalysisTemplate(clientsetinterface clusterconfig.ClientInterface, namespace string, filterOpts *filters.Options) ([]ResourceInfo, error) {
	clientsetargorollouts := clientsetinterface.GetArgoRolloutsClient()
	argoRolloutList, err := clientsetargorollouts.ArgoprojV1alpha1().Rollouts(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	argoRolloutAnalysisTemplateList, _ := clientsetargorollouts.ArgoprojV1alpha1().AnalysisTemplates(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	if err != nil {
		return nil, err
	}

	var analysisTemplateList []ResourceInfo

	for _, argoRolloutAnalysisTemplate := range argoRolloutAnalysisTemplateList.Items {
		if argoRolloutAnalysisTemplate.Labels["kor/used"] == "false" {
			reason := "Marked with unused label"
			analysisTemplateList = append(analysisTemplateList, ResourceInfo{Name: argoRolloutAnalysisTemplate.Name, Reason: reason})
			continue
		}
	}
	for _, argoRolloutAnalysisTemplate := range argoRolloutAnalysisTemplateList.Items {
		templateNameInUse := false
		for _, argoRollout := range argoRolloutList.Items {
			if pass, _ := filter.SetObject(&argoRollout).Run(filterOpts); pass {
				continue
			}

			skip := argoRollout.Spec.Strategy.Canary == nil || argoRollout.Spec.Strategy.Canary.Analysis == nil || len(argoRollout.Spec.Strategy.Canary.Analysis.Templates) < 1
			if !skip {
				rolloutCanaryAnalysis := argoRollout.Spec.Strategy.Canary.Analysis.Templates
				for _, canaryAnalysisItem := range rolloutCanaryAnalysis {
					templateNameInUse = canaryAnalysisItem.TemplateName == argoRolloutAnalysisTemplate.Name
					if templateNameInUse {
						continue
					}
				}
			}

			skip = argoRollout.Spec.Strategy.BlueGreen == nil || argoRollout.Spec.Strategy.BlueGreen.PrePromotionAnalysis == nil || len(argoRollout.Spec.Strategy.BlueGreen.PrePromotionAnalysis.Templates) < 1
			if !skip {
				rolloutBlueGreenAnalysis := argoRollout.Spec.Strategy.BlueGreen.PrePromotionAnalysis.Templates
				for _, blueGreenAnalysisAnalysisItem := range rolloutBlueGreenAnalysis {
					for _, argoRolloutAnalysisTemplate := range argoRolloutAnalysisTemplateList.Items {
						templateNameInUse = blueGreenAnalysisAnalysisItem.TemplateName == argoRolloutAnalysisTemplate.Name
						if templateNameInUse {
							continue
						}
					}
				}
			}
		}
		if !templateNameInUse {
			reason := "Argo Rollouts Analysis Templates is not in use"
			analysisTemplateList = append(analysisTemplateList, ResourceInfo{Name: argoRolloutAnalysisTemplate.Name, Reason: reason})
		}
	}
	return analysisTemplateList, nil
}

func processNamespaceArgoClusterAnalysisTemplate(clientsetinterface clusterconfig.ClientInterface, namespace string, filterOpts *filters.Options) ([]ResourceInfo, error) {
	clientset := clientsetinterface.GetKubernetesClient()
	clientsetargorollouts := clientsetinterface.GetArgoRolloutsClient()

	var clusterAnalysisTemplateList []ResourceInfo

	argoRolloutAnalysisTemplateList, _ := clientsetargorollouts.ArgoprojV1alpha1().ClusterAnalysisTemplates().List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})
	for _, argoRolloutAnalysisTemplate := range argoRolloutAnalysisTemplateList.Items {
		if argoRolloutAnalysisTemplate.Labels["kor/used"] == "false" {
			reason := "Marked with unused label"
			clusterAnalysisTemplateList = append(clusterAnalysisTemplateList, ResourceInfo{Name: argoRolloutAnalysisTemplate.Name, Reason: reason})
			continue
		}
	}

	for _, namespace := range filterOpts.Namespaces(clientset) {
		argoRolloutList, _ := clientsetargorollouts.ArgoprojV1alpha1().Rollouts(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: filterOpts.IncludeLabels})

		for _, argoRolloutAnalysisTemplate := range argoRolloutAnalysisTemplateList.Items {
			templateNameInUse := false
			for _, argoRollout := range argoRolloutList.Items {
				if pass, _ := filter.SetObject(&argoRollout).Run(filterOpts); pass {
					continue
				}

				templateNameInUse = false
				skip := argoRollout.Spec.Strategy.Canary == nil || argoRollout.Spec.Strategy.Canary.Analysis == nil || len(argoRollout.Spec.Strategy.Canary.Analysis.Templates) < 1
				if !skip {
					rolloutCanaryAnalysis := argoRollout.Spec.Strategy.Canary.Analysis.Templates
					for _, canaryAnalysisItem := range rolloutCanaryAnalysis {
						if canaryAnalysisItem.TemplateName == argoRolloutAnalysisTemplate.Name {
							templateNameInUse = true
							continue
						}

					}
				}

				skip = argoRollout.Spec.Strategy.BlueGreen == nil || argoRollout.Spec.Strategy.BlueGreen.PrePromotionAnalysis == nil || len(argoRollout.Spec.Strategy.BlueGreen.PrePromotionAnalysis.Templates) < 1
				if !skip {
					rolloutBlueGreenAnalysis := argoRollout.Spec.Strategy.BlueGreen.PrePromotionAnalysis.Templates
					for _, blueGreenAnalysisAnalysisItem := range rolloutBlueGreenAnalysis {
						for _, argoRolloutAnalysisTemplate := range argoRolloutAnalysisTemplateList.Items {
							if blueGreenAnalysisAnalysisItem.TemplateName == argoRolloutAnalysisTemplate.Name {
								templateNameInUse = true
								continue
							}
						}
					}
				}
			}
			if !templateNameInUse {
				reason := "Argo Rollouts Cluster Analysis Templates is not in use"
				if !SkipIfContainsValue(clusterAnalysisTemplateList, "Name", argoRolloutAnalysisTemplate.Name) {
					clusterAnalysisTemplateList = append(clusterAnalysisTemplateList, ResourceInfo{Name: argoRolloutAnalysisTemplate.Name, Reason: reason})
				}
			}
		}
	}

	return clusterAnalysisTemplateList, nil
}
