package main

import (
	"encoding/json"
	"fmt"
	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
	"github.com/yonahd/kor/pkg/kor"
	"k8s.io/client-go/kubernetes"
	"net/http"
)

func getUnusedConfigMapWithFilters(w http.ResponseWriter, opts common.Opts, filterOpts *filters.Options, clientset kubernetes.Interface) {
	outputFormat := "json"
	// Call the function that returns a JSON string
	response, err := kor.GetUnusedConfigmaps(filterOpts, clientset, outputFormat, opts)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		errorMsg := fmt.Sprintf("Failed to get configmaps: %v\n", err)
		json.NewEncoder(w).Encode(SimpleResponse{Message: errorMsg})
		return
	}

	// Declare a variable to hold the parsed JSON structure
	var parsedResponse map[string]interface{}

	// Parse the JSON string into a map
	if err := json.Unmarshal([]byte(response), &parsedResponse); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		errorMsg := fmt.Sprintf("Failed to parse configmaps response: %v\n", err)
		json.NewEncoder(w).Encode(SimpleResponse{Message: errorMsg})
		return
	}

	// Send the parsed JSON as the response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(parsedResponse)
}
