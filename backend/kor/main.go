package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/swaggo/http-swagger"
	"github.com/yonahd/kor/pkg/common"
	"github.com/yonahd/kor/pkg/filters"
	"github.com/yonahd/kor/pkg/kor"
	"k8s.io/client-go/kubernetes"
	"log"
	_ "main/docs"
	"net/http"
	"os"
)

var jwtSecret = []byte(os.Getenv("KOR_API_SECRET"))

type SimpleResponse struct {
	Message string `json:"message"`
}

var clientset *kubernetes.Clientset

var emptyOpts = common.Opts{
	WebhookURL:    "",
	Channel:       "",
	Token:         "",
	DeleteFlag:    false,
	NoInteractive: true,
	GroupBy:       "namespace",
}

// Auth middleware that verifies the JWT token using golang-jwt/jwt
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !(os.Getenv("NO_AUTH") == "true") {
			validateTokenAndCallNextHttpCall(w, r, next)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

// Validate list namespaces
func validateListNamespaces(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := getListNamespacesErrorIfExists(clientset, w)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			errorMsg := fmt.Sprintf("Failed to retreive namespaces: %v\n", err)
			json.NewEncoder(w).Encode(SimpleResponse{Message: errorMsg})
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

// Recovery middleware that recovers from panics and returns a 500 error
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				errorMsg := fmt.Sprintf("Internal Server Error: %v\n", err)
				json.NewEncoder(w).Encode(SimpleResponse{Message: errorMsg})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// @Summary Health Check
// @Description Returns the status of the server
// @Router /healthcheck [get]
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	_, err := clientset.Discovery().ServerVersion()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		errorMsg := fmt.Sprintf("Failure: %v\n", err)
		json.NewEncoder(w).Encode(SimpleResponse{Message: errorMsg})
		return
	}
	json.NewEncoder(w).Encode(SimpleResponse{Message: "OK"})
}

// @Summary Get Unused configmaps from all namespaces
// @Accept json
// @Produce json
// @Router /api/v1/configmaps [get]
// @Param Authorization header string false "Authorization token"
func getUnusedConfigmaps(w http.ResponseWriter, r *http.Request) {
	getUnusedConfigMapWithFilters(w, emptyOpts, &filters.Options{}, clientset)
}

// @Summary Get Unused configmaps from a specific namespace
// @Description asd
// @Accept json
// @Produce json
// @Router /api/v1/namespaces/{namespace}/configmaps [get]
// @Param Authorization header string false "Authorization token"
// @Param namespace path string true "namespace"
func getUnusedConfigmapsForNamespace(w http.ResponseWriter, r *http.Request) {
	// Extract the "namespace" parameter from the path
	namespaceArr := []string{mux.Vars(r)["namespace"]}

	getUnusedConfigMapWithFilters(w, emptyOpts, &filters.Options{
		IncludeNamespaces: namespaceArr,
	}, clientset)
}

// @title KOR API Swagger
// @version 1.0
// @description KOR API Swagger
func main() {
	router := mux.NewRouter()
	clientset = kor.GetKubeClient("")

	// Base path for the API is /api/v1
	api := router.PathPrefix("/api/v1").Subrouter()

	router.HandleFunc("/healthcheck", healthCheckHandler).Methods("GET")
	api.Handle("/configmaps", authMiddleware(validateListNamespaces(http.HandlerFunc(getUnusedConfigmaps)))).Methods("GET")
	api.Handle("/namespaces/{namespace}/configmaps", authMiddleware(http.HandlerFunc(getUnusedConfigmapsForNamespace))).Methods("GET")

	// Swagger documentation route
	router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	recoveredRouter := recoveryMiddleware(router)
	// Start HTTPS server
	srv := &http.Server{
		Addr:    "localhost:8080",
		Handler: recoveredRouter,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}
	log.Println("Server running on https://localhost:8080")
	if err := srv.ListenAndServeTLS("server.crt", "server.key"); err != nil && err != http.ErrServerClosed {
		log.Printf("Error starting server: %v", err)
	}
}
