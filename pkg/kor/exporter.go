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

	"github.com/yonahd/kor/pkg/clusterconfig"
	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
)

var (
	orphanedResourcesCounter = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kubernetes_orphaned_resources",
			Help: "Orphaned resources in Kubernetes",
		},
		[]string{"kind", "namespace", "resourceName"},
	)
)

func init() {
	prometheus.MustRegister(orphanedResourcesCounter)
}

// TODO: add option to change port / url !?
func Exporter(filterOptions *filters.Options, clientset kubernetes.Interface, apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface, clientsetinterface clusterconfig.ClientInterface, outputFormat string, opts common.Opts, resourceList []string) {
	http.Handle("/metrics", promhttp.Handler())
	fmt.Println("Server listening on :8080")
	go exportMetrics(filterOptions, clientset, apiExtClient, dynamicClient, clientsetinterface, outputFormat, opts, resourceList) // Start exporting metrics in the background
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println(err)
	}
}

func exportMetrics(filterOptions *filters.Options, clientset kubernetes.Interface, apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface, clientsetinterface clusterconfig.ClientInterface, outputFormat string, opts common.Opts, resourceList []string) {
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
		if korOutput, err := getUnusedResources(filterOptions, clientset, apiExtClient, dynamicClient, clientsetinterface, outputFormat, opts, resourceList); err != nil {
			fmt.Println(err)
			os.Exit(1)
		} else {
			var data map[string]map[string][]string
			if err := json.Unmarshal([]byte(korOutput), &data); err != nil {
				fmt.Println("Error parsing JSON:", err)
				return
			}

			orphanedResourcesCounter.Reset()

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

func getUnusedResources(filterOptions *filters.Options, clientset kubernetes.Interface, apiExtClient apiextensionsclientset.Interface, dynamicClient dynamic.Interface, clientsetinterface clusterconfig.ClientInterface, outputFormat string, opts common.Opts, resourceList []string) (string, error) {
	if len(resourceList) == 0 || (len(resourceList) == 1 && resourceList[0] == "all") {
		return GetUnusedAll(filterOptions, clientset, apiExtClient, dynamicClient, clientsetinterface, outputFormat, opts)
	}
	return GetUnusedMulti(strings.Join(resourceList, ","), filterOptions, clientset, apiExtClient, dynamicClient, clientsetinterface, outputFormat, opts)

}
