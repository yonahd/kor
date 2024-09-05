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

var jwtSecret = []byte(os.Getenv("KOR_API_SECRET"))

type SimpleResponse struct {
	Message string `json:"message"`
}

type postRequest struct {
	Data string `json:"data"`
}

var clientset *kubernetes.Clientset

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

// @Summary Health Check
// @Description Returns the status of the server
// @Success 200 {object} response
// @Router /healthcheck [get]
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(SimpleResponse{Message: "OK"})
}

// @Summary Example GET endpoint
// @Description An example GET API
// @Accept json
// @Produce json
// @Success 200 {object} response
// @Router /api/v1/configmaps [get]
// @Param Authorization header string false "Authorization token"
func getUnusedConfigmaps(w http.ResponseWriter, r *http.Request) {
	// Defer function to recover from any panic
	defer func() {
		if err := recover(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			errorMsg := fmt.Sprintf("A fatal error occurred: %v", err)
			json.NewEncoder(w).Encode(SimpleResponse{Message: errorMsg})
			log.Printf("Recovered from panic: %v", err) // Optionally log the error
		}
	}()

	// Your normal business logic
	outputFormat := "json"
	opts := common.Opts{
		WebhookURL:    "",
		Channel:       "",
		Token:         "",
		DeleteFlag:    false,
		NoInteractive: true,
		GroupBy:       "namespace",
	}

	// Try to get unused configmaps
	response, err := kor.GetUnusedConfigmaps(&filters.Options{}, clientset, outputFormat, opts)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		errorMsg := fmt.Sprintf("Failed to get configmaps: %v\n", err)
		json.NewEncoder(w).Encode(SimpleResponse{Message: errorMsg})
		return
	}

	// If successful, encode the response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// @Summary Example POST endpoint
// @Description An example POST API
// @Accept json
// @Produce json
// @Param request body postRequest true "Post Request Data"
// @Success 200 {object} response
// @Router /api/v1/example-post [post]
// @Param Authorization header string false "Authorization token"
func examplePostHandler(w http.ResponseWriter, r *http.Request) {
	var req postRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SimpleResponse{Message: "Invalid request"})
		return
	}
	json.NewEncoder(w).Encode("")
}

// @title KOR API Swagger
// @version 1.0
// @description KOR API Swagger
func main() {
	router := mux.NewRouter()
	clientset = kor.GetKubeClient("")

	router.HandleFunc("/healthcheck", healthCheckHandler).Methods("GET")
	// Base path for the API is /api/v1
	api := router.PathPrefix("/api/v1").Subrouter()
	api.Handle("/configmaps", authMiddleware(http.HandlerFunc(getUnusedConfigmaps))).Methods("GET")
	api.Handle("/example-post", authMiddleware(http.HandlerFunc(examplePostHandler))).Methods("POST")
	// Swagger documentation route
	router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	// Start HTTPS server
	srv := &http.Server{
		Addr:    "localhost:8080",
		Handler: router,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	log.Println("Server running on https://localhost:8080")
	log.Fatal(srv.ListenAndServeTLS("server.crt", "server.key"))
}
