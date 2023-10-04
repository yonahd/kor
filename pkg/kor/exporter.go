package kor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/client-go/kubernetes"
)

var (
	orphanedResourcesCounter = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
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
	fmt.Println("Server listening on :8080")
	go exportMetrics(includeExcludeLists, clientset, outputFormat) // Start exporting metrics in the background
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println(err)
	}
}

func exportMetrics(includeExcludeLists IncludeExcludeLists, clientset kubernetes.Interface, outputFormat string) {
	exporterInterval := os.Getenv("EXPORTER_INTERVAL")
	if exporterInterval == "" {
		exporterInterval = "10"
	}
	exporterIntervalValue, err := strconv.Atoi(exporterInterval)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

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
					for _, resourceName := range resourceList {
						orphanedResourcesCounter.WithLabelValues(kind, namespace, resourceName).Set(1)
					}
				}
			}
			time.Sleep(time.Duration(exporterIntervalValue) * time.Minute)
		}
	}
}
