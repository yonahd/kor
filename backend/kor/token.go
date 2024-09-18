package main

import (
	"encoding/json"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
)

func validateTokenAndCallNextHttpCall(w http.ResponseWriter, r *http.Request, next http.Handler) {
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
	}
	next.ServeHTTP(w, r)
}
