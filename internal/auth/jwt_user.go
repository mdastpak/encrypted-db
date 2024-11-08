package auth

// import (
// 	"net/http"
// 	"time"

// 	"github.com/dgrijalva/jwt-go"
// )

// // GenerateUserToken generates a JWT token for a user role
// func GenerateUserToken(username string) (string, error) {
// 	return (username, "user", time.Now().Add(15*time.Minute))
// }

// // UserMiddleware validates JWT tokens for users and checks their claims
// func UserMiddleware(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		tokenString := r.Header.Get("Authorization") // Retrieve token from Authorization header

// 		claims := &Claims{}
// 		// Parse token and validate claims
// 		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
// 			return publicKey, nil
// 		})

// 		// Check for token validity and if the role is "user"
// 		if err != nil || !token.Valid || claims.Role != "user" {
// 			http.Error(w, "Unauthorized", http.StatusUnauthorized)
// 			return
// 		}

// 		// Set user information in the context if needed (optional)
// 		// ctx := context.WithValue(r.Context(), "username", claims.Username)
// 		// r = r.WithContext(ctx)

// 		next.ServeHTTP(w, r)
// 	})
// }
