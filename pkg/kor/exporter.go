package kor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/client-go/kubernetes"
)

var (
	orphanedResourcesCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "Kubernetes_Orphaned_Resources",
			Help: "Orphaned resources in Kubernetes",
		},
		[]string{"kind", "namespace", "resourceName"},
	)
)

func init() {
	prometheus.MustRegister(orphanedResourcesCounter)
}

// TODO: add option to change port / url !?
func Exporter(includeExcludeLists IncludeExcludeLists, clientset kubernetes.Interface, outputFormat string) {
	http.Handle("/metrics", promhttp.Handler())

	go exportMetrics(includeExcludeLists, clientset, outputFormat) // Start exporting metrics in the background
	fmt.Println("Server listening on :8080")
	http.ListenAndServe(":8080", nil)
	// exportMetrics(includeExcludeLists, kubeconfig, outputFormat)
}

func exportMetrics(includeExcludeLists IncludeExcludeLists, clientset kubernetes.Interface, outputFormat string) {
	for {
		if korOutput, err := GetUnusedAllStructured(includeExcludeLists, clientset, outputFormat); err != nil {
			fmt.Println(err)
			os.Exit(1)
		} else {
			var data map[string]map[string][]string
			if err := json.Unmarshal([]byte(korOutput), &data); err != nil {
				fmt.Println("Error parsing JSON:", err)
				return
			}

			for namespace, resources := range data {
				for kind, resourceList := range resources {
					if resourceList != nil {
						for _, resourceName := range resourceList {
							// orphanedResourcesCounter.WithLabelValues(kind, namespace, resourceName).Inc()
							orphanedResourcesCounter.WithLabelValues(kind, namespace, resourceName)
						}
					}
				}
			}
			// TODO: interval
			time.Sleep(1 * time.Minute)
		}
	}
}
