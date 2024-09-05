package main

//How to use:
//export USER_SECRET=abcd12; go run generate_token.go
import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"os"
)

var jwtSecret = []byte(os.Getenv("KOR_API_SECRET")) // Replace with your actual secret key

// GenerateJWT generates a JWT token that doesn't expire
func GenerateJWT() (string, error) {
	// Create a new token object with claims (additional data)
	claims := jwt.MapClaims{
		"authorized": true,
		"user":       "exampleUser", // Replace with actual user info
		// No "exp" field means the token will not expire
	}

	// Create the token with signing method and claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with the secret key
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func main() {
	// Generate the JWT token
	token, err := GenerateJWT()
	if err != nil {
		fmt.Println("Error generating token:", err)
		return
	}

	// Print the generated token (Bearer token)
	fmt.Println(token)
}
