package kor

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	discoveryfake "k8s.io/client-go/discovery/fake"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

type GetFakeClientInterfacesForGetUnusedNamespacesTestFunc func(ctx context.Context, t *testing.T) (kubernetes.Interface, *dynamicfake.FakeDynamicClient)

func defineNewTypeEventObject(ns, name string) *eventsv1.Event {
	return &eventsv1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		ReportingController: "some-controller",
		Type:                "Warning",
	}
}

func defineServiceAccountObject(ns, name string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
}

func createEmptyNamespaceWithIgnoredByDefaultResource(ctx context.Context, t *testing.T) (kubernetes.Interface, *dynamicfake.FakeDynamicClient) {
	realClientset := fake.NewSimpleClientset()
	fakeDisc := &fakeHappyDiscovery{discoveryfake.FakeDiscovery{Fake: &realClientset.Fake}}
	clientset := &fakeClientset{Interface: realClientset, discovery: fakeDisc}
	scheme := getNamespaceTestSchema(t)

	ns1 := "test-namespace"
	namespace1 := defineNamespaceObject(ns1)
	_, err := clientset.CoreV1().Namespaces().Create(ctx, namespace1, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test namespace: %v", err)
	}

	sa1 := "default"
	serviceAccount1 := defineServiceAccountObject(ns1, sa1)
	_, err = clientset.CoreV1().ServiceAccounts(ns1).Create(ctx, serviceAccount1, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test namespace: %v", err)
	}

	listKinds := map[schema.GroupVersionResource]string{
		{Group: "apps", Version: "v1", Resource: "deployments"}:     "DeploymentList",
		{Group: "", Version: "v1", Resource: "namespaces"}:          "NamespaceList",
		{Group: "events.k8s.io", Version: "v1", Resource: "events"}: "EventList",
		{Group: "", Version: "v1", Resource: "events"}:              "EventList",
	}
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, listKinds, namespace1, serviceAccount1)

	return clientset, dynamicClient
}

func createNonEmptyNamespace(ctx context.Context, t *testing.T) (kubernetes.Interface, *dynamicfake.FakeDynamicClient) {
	realClientset := fake.NewSimpleClientset()
	fakeDisc := &fakeHappyDiscovery{discoveryfake.FakeDiscovery{Fake: &realClientset.Fake}}
	clientset := &fakeClientset{Interface: realClientset, discovery: fakeDisc}
	scheme := getNamespaceTestSchema(t)

	ns1 := "test-namespace"
	namespace1 := defineNamespaceObject(ns1)
	_, err := clientset.CoreV1().Namespaces().Create(ctx, namespace1, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test namespace: %v", err)
	}

	sa1 := "my-app"
	serviceAccount1 := defineServiceAccountObject(ns1, sa1)
	_, err = clientset.CoreV1().ServiceAccounts(ns1).Create(ctx, serviceAccount1, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test namespace: %v", err)
	}

	listKinds := map[schema.GroupVersionResource]string{
		{Group: "apps", Version: "v1", Resource: "deployments"}:     "DeploymentList",
		{Group: "", Version: "v1", Resource: "namespaces"}:          "NamespaceList",
		{Group: "events.k8s.io", Version: "v1", Resource: "events"}: "EventList",
		{Group: "", Version: "v1", Resource: "events"}:              "EventList",
	}
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, listKinds, namespace1, serviceAccount1)

	return clientset, dynamicClient
}

func createEmptyNamespace(ctx context.Context, t *testing.T) (kubernetes.Interface, *dynamicfake.FakeDynamicClient) {
	realClientset := fake.NewSimpleClientset()
	fakeDisc := &fakeHappyDiscovery{discoveryfake.FakeDiscovery{Fake: &realClientset.Fake}}
	clientset := &fakeClientset{Interface: realClientset, discovery: fakeDisc}
	scheme := getNamespaceTestSchema(t)

	ns1 := "empty-namespace"
	namespace1 := defineNamespaceObject(ns1)
	_, err := clientset.CoreV1().Namespaces().Create(ctx, namespace1, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test namespace: %v", err)
	}

	ns2 := "test-namespace"
	namespace2 := defineNamespaceObject(ns2)
	_, err = clientset.CoreV1().Namespaces().Create(ctx, namespace2, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test namespace: %v", err)
	}

	evtName := "some-random-event"
	newEventType := defineNewTypeEventObject(ns1, evtName)
	_, err = clientset.EventsV1().Events(ns1).Create(ctx, newEventType, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test event of group events.k8s.io: %v", err)
	}

	listKinds := map[schema.GroupVersionResource]string{
		{Group: "apps", Version: "v1", Resource: "deployments"}:     "DeploymentList",
		{Group: "", Version: "v1", Resource: "namespaces"}:          "NamespaceList",
		{Group: "events.k8s.io", Version: "v1", Resource: "events"}: "EventList",
		{Group: "", Version: "v1", Resource: "events"}:              "EventList",
	}
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, listKinds, namespace1, namespace2, newEventType)

	return clientset, dynamicClient
}

func createNonEmptyNamespaceLabeledAsUnused(ctx context.Context, t *testing.T) (kubernetes.Interface, *dynamicfake.FakeDynamicClient) {
	realClientset := fake.NewSimpleClientset()
	fakeDisc := &fakeHappyDiscovery{discoveryfake.FakeDiscovery{Fake: &realClientset.Fake}}
	clientset := &fakeClientset{Interface: realClientset, discovery: fakeDisc}
	scheme := getNamespaceTestSchema(t)

	ns1 := "nonempty-namespace-labeled"
	namespace1 := defineNamespaceObject(ns1)
	namespace1.Labels = map[string]string{
		"kor/used": "false",
	}
	_, err := clientset.CoreV1().Namespaces().Create(ctx, namespace1, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test namespace: %v", err)
	}

	sa1 := "my-app"
	serviceAccount1 := defineServiceAccountObject(ns1, sa1)
	_, err = clientset.CoreV1().ServiceAccounts(ns1).Create(ctx, serviceAccount1, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test namespace: %v", err)
	}

	ns2 := "test-namespace"
	namespace2 := defineNamespaceObject(ns2)
	_, err = clientset.CoreV1().Namespaces().Create(ctx, namespace2, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test namespace: %v", err)
	}

	sa2 := "another-app"
	serviceAccount2 := defineServiceAccountObject(ns2, sa2)
	_, err = clientset.CoreV1().ServiceAccounts(ns2).Create(ctx, serviceAccount2, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test namespace: %v", err)
	}

	listKinds := map[schema.GroupVersionResource]string{
		{Group: "apps", Version: "v1", Resource: "deployments"}:     "DeploymentList",
		{Group: "", Version: "v1", Resource: "namespaces"}:          "NamespaceList",
		{Group: "events.k8s.io", Version: "v1", Resource: "events"}: "EventList",
		{Group: "", Version: "v1", Resource: "events"}:              "EventList",
	}
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, listKinds, namespace1, namespace2, serviceAccount1, serviceAccount2)

	return clientset, dynamicClient
}

func createEmptyNamespaceLabeledAsUsed(ctx context.Context, t *testing.T) (kubernetes.Interface, *dynamicfake.FakeDynamicClient) {
	realClientset := fake.NewSimpleClientset()
	fakeDisc := &fakeHappyDiscovery{discoveryfake.FakeDiscovery{Fake: &realClientset.Fake}}
	clientset := &fakeClientset{Interface: realClientset, discovery: fakeDisc}
	scheme := getNamespaceTestSchema(t)

	ns1 := "empty-namespace-labeled"
	namespace1 := defineNamespaceObject(ns1)
	namespace1.Labels = map[string]string{
		"kor/used": "true",
	}
	_, err := clientset.CoreV1().Namespaces().Create(ctx, namespace1, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test namespace: %v", err)
	}

	listKinds := map[schema.GroupVersionResource]string{
		{Group: "apps", Version: "v1", Resource: "deployments"}:     "DeploymentList",
		{Group: "", Version: "v1", Resource: "namespaces"}:          "NamespaceList",
		{Group: "events.k8s.io", Version: "v1", Resource: "events"}: "EventList",
		{Group: "", Version: "v1", Resource: "events"}:              "EventList",
	}
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, listKinds, namespace1)

	return clientset, dynamicClient
}

func createKubeSystemNamespace(ctx context.Context, t *testing.T) (kubernetes.Interface, *dynamicfake.FakeDynamicClient) {
	realClientset := fake.NewSimpleClientset()
	fakeDisc := &fakeHappyDiscovery{discoveryfake.FakeDiscovery{Fake: &realClientset.Fake}}
	clientset := &fakeClientset{Interface: realClientset, discovery: fakeDisc}
	scheme := getNamespaceTestSchema(t)

	ns1 := "kube-system"
	namespace1 := defineNamespaceObject(ns1)
	_, err := clientset.CoreV1().Namespaces().Create(ctx, namespace1, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test namespace: %v", err)
	}

	listKinds := map[schema.GroupVersionResource]string{
		{Group: "apps", Version: "v1", Resource: "deployments"}:     "DeploymentList",
		{Group: "", Version: "v1", Resource: "namespaces"}:          "NamespaceList",
		{Group: "events.k8s.io", Version: "v1", Resource: "events"}: "EventList",
		{Group: "", Version: "v1", Resource: "events"}:              "EventList",
	}
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, listKinds, namespace1)

	return clientset, dynamicClient
}

func TestGetUnusedNamespaces(t *testing.T) {
	tests := []struct {
		name string

		getClientsFunc GetFakeClientInterfacesForGetUnusedNamespacesTestFunc

		filterOpts *filters.Options

		expectedOutput string
		expectedError  bool
	}{
		{
			name:           "Namespace contains only ignored by default resource types",
			getClientsFunc: createEmptyNamespace,
			filterOpts:     &filters.Options{},
			expectedOutput: `{
  "": {
    "Namespace": [
      "empty-namespace",
      "test-namespace"
    ]
  }
}`,
			expectedError: false,
		},
		{
			name:           "Namespace contains only ignored by default resource",
			getClientsFunc: createEmptyNamespaceWithIgnoredByDefaultResource,
			filterOpts:     &filters.Options{},
			expectedOutput: `{
  "": {
    "Namespace": [
      "test-namespace"
    ]
  }
}`,
			expectedError: false,
		},
		{
			name:           "Namespace contains non ignored by default resource",
			getClientsFunc: createNonEmptyNamespace,
			filterOpts:     &filters.Options{},
			expectedOutput: `{}`,
			expectedError:  false,
		},
		{
			name:           "Nonempty Namespace contains kor/used=false label",
			getClientsFunc: createNonEmptyNamespaceLabeledAsUnused,
			filterOpts:     &filters.Options{},
			expectedOutput: `{
  "": {
    "Namespace": [
      "nonempty-namespace-labeled"
    ]
  }
}`,
			expectedError: false,
		},
		{
			name:           "Empty Namespace contains kor/used=true label",
			getClientsFunc: createEmptyNamespaceLabeledAsUsed,
			filterOpts:     &filters.Options{},
			expectedOutput: `{}`,
			expectedError:  false,
		},
		{
			name:           "kube-system special Namespace",
			getClientsFunc: createKubeSystemNamespace,
			filterOpts:     &filters.Options{},
			expectedOutput: `{}`,
			expectedError:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()
			opts := common.Opts{
				WebhookURL:    "",
				Channel:       "",
				Token:         "",
				DeleteFlag:    false,
				NoInteractive: true,
				GroupBy:       "namespace",
			}

			clientset, dynamicClient := tt.getClientsFunc(ctx, t)
			got, err := GetUnusedNamespaces(ctx, tt.filterOpts, clientset, dynamicClient, "json", opts)
			if (err != nil) != tt.expectedError {
				t.Errorf("GetUnusedNamespaces() = expected error: %t, got: '%v'", tt.expectedError, err)
			}
			if got != tt.expectedOutput {
				t.Errorf("GetUnusedNamespaces() = got:\n'%s'\nwant:\n'%s'", got, tt.expectedOutput)
			}
		})
	}
}
