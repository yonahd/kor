package clusterconfig

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/argoproj/argo-rollouts/pkg/client/clientset/versioned"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type ClientInterface interface {
	GetKubernetesClient() kubernetes.Interface
	GetArgoRolloutsClient() versioned.Interface
}

type ClientSet struct {
	coreClient             *kubernetes.Clientset
	coreClientArgoRollouts *versioned.Clientset
}

func GetKubeConfigPath() string {
	home := homedir.HomeDir()
	return filepath.Join(home, ".kube", "config")
}

func GetConfig(kubeconfig string) (*rest.Config, error) {
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token"); err == nil {
		return rest.InClusterConfig()
	}

	if kubeconfig == "" {
		if configEnv := os.Getenv("KUBECONFIG"); configEnv != "" {
			kubeconfig = configEnv
		} else {
			kubeconfig = GetKubeConfigPath()
		}
	}

	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}

func GetKubeClient(kubeconfig string) *kubernetes.Clientset {
	config, err := GetConfig(kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load kubeconfig: %v\n", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create Kubernetes client: %v\n", err)
		os.Exit(1)
	}
	return clientset
}

func GetAPIExtensionsClient(kubeconfig string) *apiextensionsclientset.Clientset {
	config, err := GetConfig(kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load kubeconfig: %v\n", err)
		os.Exit(1)
	}

	clientset, err := apiextensionsclientset.NewForConfig(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create Kubernetes client: %v\n", err)
		os.Exit(1)
	}
	return clientset
}

func GetDynamicClient(kubeconfig string) *dynamic.DynamicClient {
	config, err := GetConfig(kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load kubeconfig: %v\n", err)
		os.Exit(1)
	}

	clientset, err := dynamic.NewForConfig(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create Kubernetes client: %v\n", err)
		os.Exit(1)
	}
	return clientset
}

func (c ClientSet) GetArgoRolloutsClient() versioned.Interface {
	return c.coreClientArgoRollouts
}

// GetKubernetesClient returns the Kubernetes core client.
func (c ClientSet) GetKubernetesClient() kubernetes.Interface {
	return c.coreClient
}

func GetKubeClientForCrds(kubeconfig string, clientset *kubernetes.Clientset) (ClientInterface, error) {
	config, err := GetConfig(kubeconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load kubeconfig: %v\n", err)
		os.Exit(1)
	}
	clientsetall, err := NewClientSetForCrd(config, clientset)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create Kubernetes client: %v\n", err)
		os.Exit(1)
	}
	return clientsetall, nil
}

func NewClientSetForCrd(config *rest.Config, clientset *kubernetes.Clientset) (ClientInterface, error) {
	// Create the custom v1 client
	coreClientArgoRolloutsV1Client, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Argo Rollouts client: %v", err)
	}

	// Return the ClientSet struct
	return &ClientSet{
		coreClient:             clientset,
		coreClientArgoRollouts: coreClientArgoRolloutsV1Client,
	}, nil
}
