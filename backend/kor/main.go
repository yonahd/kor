package main

import (
	"crypto/tls"
	"encoding/json"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/swaggo/http-swagger"
	"log"
	_ "main/docs"
	"net/http"
	"os"
)

var jwtSecret = []byte(os.Getenv("KOR_API_SECRET"))

type response struct {
	Message string `json:"message"`
}

type postRequest struct {
	Data string `json:"data"`
}

// Auth middleware that verifies the JWT token using golang-jwt/jwt
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !(os.Getenv("NO_AUTH") == "true") {
			tokenHeader := r.Header.Get("Authorization")
			if tokenHeader == "" {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(response{Message: "Missing token"})
				return
			}

			tokenString := tokenHeader[len("Bearer "):]
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				return jwtSecret, nil
			})

			if err != nil || !token.Valid {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(response{Message: "Invalid token"})
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
	json.NewEncoder(w).Encode(response{Message: "OK"})
}

// @Summary Example GET endpoint
// @Description An example GET API
// @Accept json
// @Produce json
// @Success 200 {object} response
// @Router /api/v1/example-get [get]
// @Param Authorization header string false "Authorization token"
func exampleGetHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(response{Message: "This is a GET response"})
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
		json.NewEncoder(w).Encode(response{Message: "Invalid request"})
		return
	}
	json.NewEncoder(w).Encode(response{Message: "Received: " + req.Data})
}

// @title KOR API Swagger
// @version 1.0
// @description KOR API Swagger
func main() {
	router := mux.NewRouter()

	// Base path for the API is /api/v1
	router.HandleFunc("/healthcheck", healthCheckHandler).Methods("GET")
	api := router.PathPrefix("/api/v1").Subrouter()
	api.Handle("/example-get", authMiddleware(http.HandlerFunc(exampleGetHandler))).Methods("GET")
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
