package kor

import (
	"context"
	"strings"
	"testing"

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gatewayfake "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/fake"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

func createTestGateways(t *testing.T) (*fake.Clientset, *gatewayfake.Clientset) {
	clientset := fake.NewSimpleClientset()
	gatewayClientset := gatewayfake.NewSimpleClientset()

	// Create a test namespace
	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "test-namespace"},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating test namespace: %v", err)
	}

	// Create a GatewayClass
	gatewayClass := &gatewayv1.GatewayClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-gateway-class",
		},
		Spec: gatewayv1.GatewayClassSpec{
			ControllerName: "test-controller",
		},
	}
	_, err = gatewayClientset.GatewayV1().GatewayClasses().Create(context.TODO(), gatewayClass, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating test GatewayClass: %v", err)
	}

	// Create Gateways for testing
	// Gateway 1: References existing GatewayClass but has no routes
	gateway1 := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "unused-gateway-no-routes",
			Namespace: "test-namespace",
		},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: "test-gateway-class",
			Listeners: []gatewayv1.Listener{
				{
					Name:     "http",
					Protocol: gatewayv1.HTTPProtocolType,
					Port:     80,
				},
			},
		},
	}

	// Gateway 2: References non-existing GatewayClass
	gateway2 := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "unused-gateway-missing-class",
			Namespace: "test-namespace",
		},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: "non-existent-gateway-class",
			Listeners: []gatewayv1.Listener{
				{
					Name:     "http",
					Protocol: gatewayv1.HTTPProtocolType,
					Port:     80,
				},
			},
		},
	}

	// Gateway 3: References existing GatewayClass and has routes attached
	gateway3 := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "used-gateway-with-routes",
			Namespace: "test-namespace",
		},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: "test-gateway-class",
			Listeners: []gatewayv1.Listener{
				{
					Name:     "http",
					Protocol: gatewayv1.HTTPProtocolType,
					Port:     80,
				},
			},
		},
	}

	// Gateway 4: Marked as unused with label
	gateway4 := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "labeled-unused-gateway",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"kor/used": "false",
			},
		},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: "test-gateway-class",
			Listeners: []gatewayv1.Listener{
				{
					Name:     "http",
					Protocol: gatewayv1.HTTPProtocolType,
					Port:     80,
				},
			},
		},
	}

	gateways := []*gatewayv1.Gateway{gateway1, gateway2, gateway3, gateway4}
	for _, gw := range gateways {
		_, err := gatewayClientset.GatewayV1().Gateways(gw.Namespace).Create(context.TODO(), gw, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("Error creating test Gateway %s: %v", gw.Name, err)
		}
	}

	// Create an HTTPRoute that references gateway3
	httpRoute := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-http-route",
			Namespace: "test-namespace",
		},
		Spec: gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{
						Name: "used-gateway-with-routes",
					},
				},
			},
			Rules: []gatewayv1.HTTPRouteRule{
				{
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name: "test-service",
									Port: &[]gatewayv1.PortNumber{8080}[0],
								},
							},
						},
					},
				},
			},
		},
	}

	_, err = gatewayClientset.GatewayV1().HTTPRoutes("test-namespace").Create(context.TODO(), httpRoute, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating test HTTPRoute: %v", err)
	}

	return clientset, gatewayClientset
}

func TestProcessNamespaceGateways(t *testing.T) {
	clientset, gatewayClientset := createTestGateways(t)

	unusedGateways, err := processNamespaceGateways(clientset, gatewayClientset, "test-namespace", &filters.Options{}, common.Opts{})
	if err != nil {
		t.Fatalf("Error retrieving unused Gateways: %v", err)
	}

	if len(unusedGateways) != 3 { // gateway1, gateway2, gateway4
		t.Errorf("Expected 3 unused Gateway objects, got %d", len(unusedGateways))
	}

	expectedUnused := map[string]string{
		"unused-gateway-no-routes":     "Gateway has no attached routes",
		"unused-gateway-missing-class": "Gateway references a non-existing GatewayClass",
		"labeled-unused-gateway":       "Marked with unused label",
	}

	for _, gw := range unusedGateways {
		expectedReason, exists := expectedUnused[gw.Name]
		if !exists {
			t.Errorf("Unexpected unused gateway: %s", gw.Name)
		} else if gw.Reason != expectedReason {
			t.Errorf("Gateway %s: expected reason '%s', got '%s'", gw.Name, expectedReason, gw.Reason)
		}
	}
}

func TestCheckGatewayClassExists(t *testing.T) {
	_, gatewayClientset := createTestGateways(t)

	// Test existing GatewayClass
	exists, err := checkGatewayClassExists(gatewayClientset, "test-gateway-class")
	if err != nil {
		t.Fatalf("Error checking existing GatewayClass: %v", err)
	}
	if !exists {
		t.Error("Expected GatewayClass to exist")
	}

	// Test non-existing GatewayClass
	exists, err = checkGatewayClassExists(gatewayClientset, "non-existent-gateway-class")
	if err != nil {
		t.Fatalf("Error checking non-existing GatewayClass: %v", err)
	}
	if exists {
		t.Error("Expected GatewayClass not to exist")
	}
}

func TestCheckGatewayHasRoutes(t *testing.T) {
	_, gatewayClientset := createTestGateways(t)

	// Get gateways for testing
	gateways, err := gatewayClientset.GatewayV1().Gateways("test-namespace").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("Error listing gateways: %v", err)
	}

	gatewayMap := make(map[string]*gatewayv1.Gateway)
	for _, gw := range gateways.Items {
		gatewayMap[gw.Name] = &gw
	}

	// Test gateway with routes
	hasRoutes, err := checkGatewayHasRoutes(gatewayClientset, gatewayMap["used-gateway-with-routes"])
	if err != nil {
		t.Fatalf("Error checking gateway with routes: %v", err)
	}
	if !hasRoutes {
		t.Error("Expected gateway to have routes")
	}

	// Test gateway without routes
	hasRoutes, err = checkGatewayHasRoutes(gatewayClientset, gatewayMap["unused-gateway-no-routes"])
	if err != nil {
		t.Fatalf("Error checking gateway without routes: %v", err)
	}
	if hasRoutes {
		t.Error("Expected gateway not to have routes")
	}
}

func TestGetUnusedGatewaysStructured(t *testing.T) {
	clientset, gatewayClientset := createTestGateways(t)

	opts := common.Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	output, err := GetUnusedGateways(&filters.Options{}, clientset, gatewayClientset, "json", opts)
	if err != nil {
		t.Fatalf("Error calling GetUnusedGateways: %v", err)
	}

	expectedOutputKeys := []string{"unused-gateway-no-routes", "unused-gateway-missing-class", "labeled-unused-gateway"}

	// Check if all expected gateways are in the output
	for _, expectedKey := range expectedOutputKeys {
		if !strings.Contains(output, expectedKey) {
			t.Errorf("Expected output to contain gateway: %s", expectedKey)
		}
	}
}

func TestProcessNamespaceGatewaysTCPRoute(t *testing.T) {
	_, gatewayClientset := createTestGateways(t)

	// Create a gateway with TCPRoute
	tcpGateway := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tcp-gateway",
			Namespace: "test-namespace",
		},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: "test-gateway-class",
			Listeners: []gatewayv1.Listener{
				{
					Name:     "tcp",
					Protocol: gatewayv1.TCPProtocolType,
					Port:     3306,
				},
			},
		},
	}

	_, err := gatewayClientset.GatewayV1().Gateways("test-namespace").Create(context.TODO(), tcpGateway, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating TCP Gateway: %v", err)
	}

	// Create a TCPRoute that references the TCP gateway
	tcpRoute := &gatewayv1alpha2.TCPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-tcp-route",
			Namespace: "test-namespace",
		},
		Spec: gatewayv1alpha2.TCPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{
						Name: "tcp-gateway",
					},
				},
			},
			Rules: []gatewayv1alpha2.TCPRouteRule{
				{
					BackendRefs: []gatewayv1alpha2.BackendRef{
						{
							BackendObjectReference: gatewayv1.BackendObjectReference{
								Name: "tcp-service",
								Port: &[]gatewayv1.PortNumber{3306}[0],
							},
						},
					},
				},
			},
		},
	}

	_, err = gatewayClientset.GatewayV1alpha2().TCPRoutes("test-namespace").Create(context.TODO(), tcpRoute, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating TCPRoute: %v", err)
	}

	// Test that the TCP gateway is not marked as unused
	hasRoutes, err := checkGatewayHasRoutes(gatewayClientset, tcpGateway)
	if err != nil {
		t.Fatalf("Error checking TCP gateway routes: %v", err)
	}
	if !hasRoutes {
		t.Error("Expected TCP gateway to have routes")
	}
}

func TestFilterOwnerReferencedGateways(t *testing.T) {
	clientset, gatewayClientset := createTestGateways(t)

	// Create a gateway with owner references
	ownedGateway := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "owned-gateway",
			Namespace: "test-namespace",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "some-controller",
					UID:        "some-uid",
				},
			},
		},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: "non-existent-gateway-class", // This would normally make it unused
			Listeners: []gatewayv1.Listener{
				{
					Name:     "http",
					Protocol: gatewayv1.HTTPProtocolType,
					Port:     80,
				},
			},
		},
	}

	_, err := gatewayClientset.GatewayV1().Gateways("test-namespace").Create(context.TODO(), ownedGateway, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating owned Gateway: %v", err)
	}

	// Test without owner reference filtering - should include the owned gateway
	filterOptsNoSkip := &filters.Options{IgnoreOwnerReferences: false}
	unusedWithoutFilter, err := processNamespaceGateways(clientset, gatewayClientset, "test-namespace", filterOptsNoSkip, common.Opts{})
	if err != nil {
		t.Fatalf("Error retrieving unused Gateways without filter: %v", err)
	}

	found := false
	for _, gw := range unusedWithoutFilter {
		if gw.Name == "owned-gateway" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected owned gateway to be included without owner reference filter")
	}

	// Test with owner reference filtering - should exclude the owned gateway
	filterOptsWithSkip := &filters.Options{IgnoreOwnerReferences: true}
	unusedWithFilter, err := processNamespaceGateways(clientset, gatewayClientset, "test-namespace", filterOptsWithSkip, common.Opts{})
	if err != nil {
		t.Fatalf("Error retrieving unused Gateways with filter: %v", err)
	}

	found = false
	for _, gw := range unusedWithFilter {
		if gw.Name == "owned-gateway" {
			found = true
			break
		}
	}
	if found {
		t.Error("Expected owned gateway to be excluded with owner reference filter")
	}
}