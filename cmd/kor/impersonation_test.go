package kor

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// Run the real CLI in a subprocess to isolate Cobra flags and package globals.
func TestImpersonationCLIHelper(t *testing.T) {
	if os.Getenv("KOR_TEST_IMPERSONATION_HELPER") != "1" {
		return
	}
	separator := slices.Index(os.Args, "--")
	if separator == -1 {
		t.Fatal("missing CLI argument separator")
	}
	os.Args = append([]string{"kor"}, os.Args[separator+1:]...)
	Execute()
}

func TestImpersonationCLI(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		user string
	}{
		{name: "kubeconfig identity", user: "configured-user"},
		{name: "as flag", args: []string{"--as", "admin"}, user: "admin"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			requests := make(map[string]bool)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				requests[r.Method+" "+r.URL.Path] = true
				mu.Unlock()
				if got := r.Header.Get("Impersonate-User"); got != tc.user {
					t.Errorf("%s %s: impersonated user = %q, want %q", r.Method, r.URL.Path, got, tc.user)
				}
				w.Header().Set("Content-Type", "application/json")
				var response string
				switch r.URL.Path {
				case "/api":
					response = `{"apiVersion":"v1","kind":"APIVersions","versions":["v1"]}`
				case "/apis":
					response = `{"apiVersion":"v1","kind":"APIGroupList","groups":[]}`
				case "/api/v1":
					response = `{"apiVersion":"v1","kind":"APIResourceList","groupVersion":"v1","resources":[{"name":"persistentvolumeclaims","singularName":"persistentvolumeclaim","namespaced":true,"kind":"PersistentVolumeClaim","verbs":["get","list","delete"],"shortNames":["pvc"]}]}`
				case "/api/v1/namespaces/test":
					response = `{"apiVersion":"v1","kind":"Namespace","metadata":{"name":"test"}}`
				case "/api/v1/namespaces/test/persistentvolumeclaims":
					response = `{"apiVersion":"v1","kind":"PersistentVolumeClaimList","items":[{"metadata":{"name":"unused","namespace":"test"}}]}`
				case "/api/v1/namespaces/test/pods":
					response = `{"apiVersion":"v1","kind":"PodList","items":[]}`
				case "/api/v1/namespaces/test/persistentvolumeclaims/unused":
					response = `{"apiVersion":"v1","kind":"Status","status":"Success"}`
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
			config.Clusters["test"] = &clientcmdapi.Cluster{Server: server.URL}
			config.AuthInfos["test"] = &clientcmdapi.AuthInfo{Impersonate: "configured-user"}
			config.Contexts["test"] = &clientcmdapi.Context{Cluster: "test", AuthInfo: "test"}
			config.CurrentContext = "test"
			path := filepath.Join(t.TempDir(), "kubeconfig")
			if err := clientcmd.WriteToFile(*config, path); err != nil {
				t.Fatal(err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			args := []string{"-test.run=^TestImpersonationCLIHelper$"}
			// Include CLI subprocess coverage in the parent test report.
			if coverDir := flag.Lookup("test.gocoverdir"); coverDir != nil && coverDir.Value.String() != "" {
				args = append(args, "-test.gocoverdir="+coverDir.Value.String())
			}
			args = append(args, "--", "pvc", "--kubeconfig", path, "--include-namespaces", "test", "--delete", "--no-interactive")
			cmd := exec.CommandContext(ctx, os.Args[0], append(args, tc.args...)...)
			cmd.Env = append(os.Environ(), "KOR_TEST_IMPERSONATION_HELPER=1", "SLACK_WEBHOOK_URL=", "SLACK_AUTH_TOKEN=", "SLACK_CHANNEL=")
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("CLI failed: %v\n%s", err, output)
			}

			mu.Lock()
			defer mu.Unlock()
			for _, request := range []string{
				"GET /api", "GET /apis", "GET /api/v1",
				"GET /api/v1/namespaces/test/persistentvolumeclaims",
				"DELETE /api/v1/namespaces/test/persistentvolumeclaims/unused",
			} {
				if !requests[request] {
					t.Errorf("missing request: %s", request)
				}
			}
		})
	}
}
