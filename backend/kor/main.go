package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
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
// move configmap functions to a separate configmap.go file
// emptyOpts is a common value - separate file
// jwtSecret is a common value - separate file

var jwtSecret = []byte(os.Getenv("KOR_API_SECRET"))

type SimpleResponse struct {
	Message string `json:"message"`
}

var clientset *kubernetes.Clientset

emptyOpts := common.Opts{
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
			tokenHeader := r.Header.Get("Authorization")
			if tokenHeader == "" {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(SimpleResponse{Message: "Missing token"})
				return
			}

			tokenString := tokenHeader[len("Bearer "):]
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				return jwtSecret, nil
			})

			if err != nil || !token.Valid {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(SimpleResponse{Message: "Invalid token"})
				return
			}

			next.ServeHTTP(w, r)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

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

	opts := common.Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	getUnusedConfigMapWithFilters(w, opts, &filters.Options{})
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

	getUnusedConfigMapWithFilters(w, opts, &filters.Options{
		IncludeNamespaces: namespaceArr,
	})
}

func getUnusedConfigMapWithFilters(w http.ResponseWriter, opts common.Opts, filterOpts *filters.Options) {
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

// @title KOR API Swagger
// @version 1.0
// @description KOR API Swagger
func main() {
	router := mux.NewRouter()
	clientset = kor.GetKubeClient("")

	// Base path for the API is /api/v1
	api := router.PathPrefix("/api/v1").Subrouter()

	router.HandleFunc("/healthcheck", healthCheckHandler).Methods("GET")
	api.Handle("/configmaps", authMiddleware(http.HandlerFunc(getUnusedConfigmaps))).Methods("GET")
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
