package kor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

var (
	orphanedResourcesGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kubernetes_orphaned_resources",
			Help: "Orphaned resources in Kubernetes",
		},
		[]string{"kind", "namespace", "resourceName"},
	)
)

func init() {
	prometheus.MustRegister(orphanedResourcesGauge)
}

// TODO: add option to change port / url !?
func Exporter(filterOptions *filters.Options, clientset kubernetes.Interface, apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface, outputFormat string, opts common.Opts, resourceList []string) {
	http.Handle("/metrics", promhttp.Handler())
	fmt.Println("Server listening on :8080")
	go exportMetrics(filterOptions, clientset, apiExtClient, dynamicClient, outputFormat, opts, resourceList) // Start exporting metrics in the background
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println(err)
	}
}

func exportMetrics(filterOptions *filters.Options, clientset kubernetes.Interface, apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface, outputFormat string, opts common.Opts, resourceList []string) {
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
		fmt.Println("collecting unused resources")
		if korOutput, err := getUnusedResources(filterOptions, clientset, apiExtClient, dynamicClient, outputFormat, opts, resourceList); err != nil {
			fmt.Println(err)
			os.Exit(1)
		} else {
			var data map[string]map[string][]string
			if err := json.Unmarshal([]byte(korOutput), &data); err != nil {
				fmt.Println("Error parsing JSON:", err)
				return
			}

			orphanedResourcesGauge.Reset()
			setOrphanedResourceMetrics(data, opts.GroupBy)
			time.Sleep(time.Duration(exporterIntervalValue) * time.Minute)
		}
	}
}

func setOrphanedResourceMetrics(data map[string]map[string][]string, groupBy string) {
	labelValues := func(outerKey, innerKey string) (kind, namespace string) {
		return innerKey, outerKey
	}
	if groupBy == "resource" {
		labelValues = func(outerKey, innerKey string) (kind, namespace string) {
			return outerKey, innerKey
		}
	}

	for outerKey, resources := range data {
		for innerKey, resourceList := range resources {
			for _, resourceName := range resourceList {
				kind, namespace := labelValues(outerKey, innerKey)
				orphanedResourcesGauge.WithLabelValues(kind, namespace, resourceName).Set(1)
			}
		}
	}
}

func getUnusedResources(filterOptions *filters.Options, clientset kubernetes.Interface, apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface, outputFormat string, opts common.Opts, resourceList []string) (string, error) {
	if len(resourceList) == 0 || (len(resourceList) == 1 && resourceList[0] == "all") {
		return GetUnusedAll(filterOptions, clientset, apiExtClient, dynamicClient, outputFormat, opts)
	}
	return GetUnusedMulti(strings.Join(resourceList, ","), filterOptions, clientset, apiExtClient, dynamicClient, outputFormat, opts)

}
