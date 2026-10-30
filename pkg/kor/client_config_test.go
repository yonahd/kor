package kor

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func TestClientImpersonation(t *testing.T) {
	t.Cleanup(func() { SetImpersonatedUser("") })

	kubeconfigIdentity := rest.ImpersonationConfig{
		UserName: "configured-user",
		UID:      "configured-uid",
		Groups:   []string{"configured-group"},
		Extra:    map[string][]string{"project": {"example"}},
	}
	overriddenIdentity := kubeconfigIdentity
	overriddenIdentity.UserName = "admin"

	for _, tc := range []struct {
		name       string
		configured rest.ImpersonationConfig
		user       string
		want       rest.ImpersonationConfig
		useEnv     bool
	}{
		{name: "no impersonation"},
		{name: "explicit user", user: "admin", want: rest.ImpersonationConfig{UserName: "admin"}},
		{name: "empty user restores kubeconfig identity", configured: kubeconfigIdentity, want: kubeconfigIdentity},
		{name: "override kubeconfig username", configured: kubeconfigIdentity, user: "admin", want: overriddenIdentity},
		{name: "KUBECONFIG environment", configured: kubeconfigIdentity, user: "admin", want: overriddenIdentity, useEnv: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			SetImpersonatedUser(tc.user)

			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("Impersonate-User"); got != tc.want.UserName {
					t.Errorf("%s %s: impersonated user = %q, want %q", r.Method, r.URL.Path, got, tc.want.UserName)
				}
				if got := r.Header.Get("Impersonate-Uid"); got != tc.want.UID {
					t.Errorf("impersonated UID = %q, want %q", got, tc.want.UID)
				}
				if got := r.Header.Values("Impersonate-Group"); !reflect.DeepEqual(got, tc.want.Groups) {
					t.Errorf("impersonated groups = %v, want %v", got, tc.want.Groups)
				}
				if got := r.Header.Values("Impersonate-Extra-Project"); !reflect.DeepEqual(got, tc.want.Extra["project"]) {
					t.Errorf("impersonated extras = %v, want %v", got, tc.want.Extra["project"])
				}
				if got := r.Header.Get("Authorization"); got != "Bearer original-token" {
					t.Errorf("original credentials not preserved: Authorization = %q", got)
				}

				w.Header().Set("Content-Type", "application/json")
				var response string
				switch {
				case r.URL.Path == "/version":
					response = `{"major":"1","minor":"37","gitVersion":"v1.37.0"}`
				case r.Method == http.MethodDelete:
					response = `{"apiVersion":"v1","kind":"Status","status":"Success"}`
				case r.URL.Path == "/api/v1/namespaces/test/secrets":
					response = `{"apiVersion":"v1","kind":"SecretList","items":[]}`
				case r.URL.Path == "/apis/apiextensions.k8s.io/v1/customresourcedefinitions":
					response = `{"apiVersion":"apiextensions.k8s.io/v1","kind":"CustomResourceDefinitionList","items":[]}`
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if _, err := fmt.Fprint(w, response); err != nil {
					t.Errorf("write response: %v", err)
				}
			}))
			defer server.Close()

			config := clientcmdapi.NewConfig()
			config.Clusters["test"] = &clientcmdapi.Cluster{Server: server.URL, InsecureSkipTLSVerify: true}
			config.AuthInfos["test"] = &clientcmdapi.AuthInfo{
				Token:                "original-token",
				Impersonate:          tc.configured.UserName,
				ImpersonateUID:       tc.configured.UID,
				ImpersonateGroups:    tc.configured.Groups,
				ImpersonateUserExtra: tc.configured.Extra,
			}
			config.Contexts["test"] = &clientcmdapi.Context{Cluster: "test", AuthInfo: "test"}
			config.CurrentContext = "test"
			path := filepath.Join(t.TempDir(), "kubeconfig")
			if err := clientcmd.WriteToFile(*config, path); err != nil {
				t.Fatal(err)
			}
			if tc.useEnv {
				t.Setenv("KUBECONFIG", path)
				path = ""
			}

			typedClient := GetKubeClient(path)
			if _, err := typedClient.Discovery().ServerVersion(); err != nil {
				t.Fatalf("discovery: %v", err)
			}
			if _, err := typedClient.CoreV1().Secrets("test").List(context.Background(), metav1.ListOptions{}); err != nil {
				t.Fatalf("typed list: %v", err)
			}
			if err := typedClient.CoreV1().Secrets("test").Delete(context.Background(), "unused", metav1.DeleteOptions{}); err != nil {
				t.Fatalf("typed delete: %v", err)
			}
			if _, err := GetAPIExtensionsClient(path).ApiextensionsV1().CustomResourceDefinitions().List(context.Background(), metav1.ListOptions{}); err != nil {
				t.Fatalf("API extensions list: %v", err)
			}
			gvr := schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "examples"}
			if err := GetDynamicClient(path).Resource(gvr).Namespace("test").Delete(context.Background(), "unused", metav1.DeleteOptions{}); err != nil {
				t.Fatalf("dynamic delete: %v", err)
			}
		})
	}
}

func TestGetConfigInvalidPathWithImpersonation(t *testing.T) {
	SetImpersonatedUser("admin")
	t.Cleanup(func() { SetImpersonatedUser("") })
	config, err := GetConfig(filepath.Join(t.TempDir(), "missing"))
	if err == nil || config != nil {
		t.Fatalf("expected nil config and an error, got %v, %v", config, err)
	}
}
